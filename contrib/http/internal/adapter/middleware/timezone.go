//go:build http

// timezone.go
//
// TimezoneMiddleware resolves the operator's preferred display timezone once
// per request and stashes a *time.Location on the request context. Views and
// view adapters then call LocationFromContext(ctx) without re-fetching the
// user row.
//
// The timezone context key and location helpers are duplicated here (from
// pyeza/types/timezone.go) to avoid adding a pyeza dependency to espyna.
// The caller's context key must match what downstream consumers read -- if the
// consumer reads via pyeza's LocationFromContext, the wiring layer must bridge
// the two context keys (or use the same key type, which is not possible across
// packages). The recommended approach is for the wiring layer to pass a
// WithLocation function that stores on the pyeza key.
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultTimezone is the fallback used when no other zone is known.
const DefaultTimezone = "Asia/Manila"

// tzContextKey is the typed context key for the timezone location.
type tzContextKey struct{}

// loadLocationOrDefault returns the *time.Location for name, or the
// DefaultTimezone location, or time.UTC as a last resort.
func loadLocationOrDefault(name string) *time.Location {
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	if loc, err := time.LoadLocation(DefaultTimezone); err == nil {
		return loc
	}
	return time.UTC
}

// WithTimezoneLocation stores tz on ctx under the timezone key.
func WithTimezoneLocation(ctx context.Context, tz *time.Location) context.Context {
	if tz == nil {
		return ctx
	}
	return context.WithValue(ctx, tzContextKey{}, tz)
}

// TimezoneLocationFromContext returns the *time.Location stored via
// WithTimezoneLocation. Falls back to DefaultTimezone if absent.
func TimezoneLocationFromContext(ctx context.Context) *time.Location {
	if ctx != nil {
		if loc, ok := ctx.Value(tzContextKey{}).(*time.Location); ok && loc != nil {
			return loc
		}
	}
	return loadLocationOrDefault("")
}

// isStaticAsset returns true for paths that serve pre-built assets and
// therefore need no per-request timezone resolution.
func isStaticAsset(path string) bool {
	return strings.HasPrefix(path, "/assets/") || path == "/favicon.ico"
}

// UserTimezoneLookupFunc resolves a user's timezone preference from their
// profile. Returns the IANA timezone name (e.g. "Asia/Manila") or "" if
// unknown. The request context is provided for DB access.
type UserTimezoneLookupFunc func(ctx context.Context, userID string) (string, error)

// UserIDFromContextFunc extracts the authenticated user ID from the request
// context. Returns "" when no user is authenticated.
type UserIDFromContextFunc func(ctx context.Context) string

// TimezoneMiddleware resolves the operator's preferred display timezone once
// per request and stashes a *time.Location on the request context under the
// shared timezone key.
//
// The UserIDFromContext and LookupUserTZ functions are injected to decouple
// from the consumer package (avoiding import cycles). The wiring layer
// passes consumer.GetUserIDFromContext and a closure around ReadUser.Execute.
type TimezoneMiddleware struct {
	// UserIDFromContext extracts the user ID from the request context.
	// Required. Typically wired to consumer.GetUserIDFromContext.
	UserIDFromContext UserIDFromContextFunc

	// LookupUserTZ reads the user's timezone preference from the DB.
	// When nil, timezone resolution is skipped (falls back to default).
	LookupUserTZ UserTimezoneLookupFunc

	// WithLocation is the function used to store the timezone on the context.
	// When nil, defaults to WithTimezoneLocation. The wiring layer can override
	// this to store on a different context key (e.g. pyeza's key).
	WithLocation func(ctx context.Context, loc *time.Location) context.Context

	// One-shot per (uid) cache to avoid hammering the user repo.
	mu    sync.RWMutex
	cache map[string]string
}

// NewTimezoneMiddleware creates a new TimezoneMiddleware with the given
// user ID extractor and timezone lookup function.
func NewTimezoneMiddleware(
	userIDFromCtx UserIDFromContextFunc,
	lookupUserTZ UserTimezoneLookupFunc,
) *TimezoneMiddleware {
	return &TimezoneMiddleware{
		UserIDFromContext: userIDFromCtx,
		LookupUserTZ:     lookupUserTZ,
		cache:            make(map[string]string),
	}
}

// Handle returns the middleware handler function.
func (m *TimezoneMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStaticAsset(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		var userID string
		if m.UserIDFromContext != nil {
			userID = m.UserIDFromContext(r.Context())
		}

		userTZ := m.lookupUserTZ(r.Context(), userID)
		loc := loadLocationOrDefault(userTZ)

		withLoc := m.WithLocation
		if withLoc == nil {
			withLoc = WithTimezoneLocation
		}
		ctx := withLoc(r.Context(), loc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *TimezoneMiddleware) lookupUserTZ(ctx context.Context, userID string) string {
	if userID == "" || m.LookupUserTZ == nil {
		return ""
	}

	m.mu.RLock()
	if tz, ok := m.cache[userID]; ok {
		m.mu.RUnlock()
		return tz
	}
	m.mu.RUnlock()

	tz, err := m.LookupUserTZ(ctx, userID)
	if err != nil {
		log.Printf("[tz-mw] read user failed uid=%s err=%v", userID, err)
		return ""
	}

	m.mu.Lock()
	m.cache[userID] = tz
	m.mu.Unlock()
	return tz
}

// InvalidateUserCache drops the cached TZ for userID so the next request
// re-reads. Call this from the user-update handler after an operator changes
// their timezone preference.
func (m *TimezoneMiddleware) InvalidateUserCache(userID string) {
	m.mu.Lock()
	delete(m.cache, userID)
	m.mu.Unlock()
}
