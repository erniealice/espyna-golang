package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TimezoneConfig configures the Timezone middleware.
type TimezoneConfig struct {
	// GetUserID extracts the authenticated user ID from the request context.
	// Required.
	GetUserID func(ctx context.Context) string

	// LookupTimezone fetches the user's timezone preference (an IANA name)
	// given their user ID and request context. Return "" to fall back to
	// DefaultTimezone.
	LookupTimezone func(ctx context.Context, userID string) string

	// StoreLocation puts the resolved *time.Location on the request context.
	// Required. The app wires this to its shared context-key helper (e.g.
	// pyezatypes.WithLocation).
	StoreLocation func(ctx context.Context, loc *time.Location) context.Context

	// DefaultTimezone is the IANA timezone name used when the user has no
	// preference or the lookup fails. Defaults to "Asia/Manila".
	DefaultTimezone string
}

// timezoneMiddleware resolves the operator's preferred display timezone once
// per request and stashes a *time.Location on the request context.
type timezoneMiddleware struct {
	cfg        TimezoneConfig
	defaultLoc *time.Location
	mu         sync.RWMutex
	cache      map[string]string
}

// Timezone returns a MiddlewareFunc that resolves the authenticated user's
// timezone preference and stores the resulting *time.Location on the request
// context. Non-authenticated requests and static asset paths pass through
// without a timezone lookup.
func Timezone(cfg TimezoneConfig) MiddlewareFunc {
	if cfg.DefaultTimezone == "" {
		cfg.DefaultTimezone = "Asia/Manila"
	}
	defaultLoc, err := time.LoadLocation(cfg.DefaultTimezone)
	if err != nil {
		defaultLoc = time.UTC
	}
	m := &timezoneMiddleware{
		cfg:        cfg,
		defaultLoc: defaultLoc,
		cache:      make(map[string]string),
	}
	return m.wrap
}

func (m *timezoneMiddleware) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isTimezoneStaticAsset(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		var userID string
		if m.cfg.GetUserID != nil {
			userID = m.cfg.GetUserID(r.Context())
		}

		loc := m.resolve(r.Context(), userID)
		if m.cfg.StoreLocation != nil {
			ctx := m.cfg.StoreLocation(r.Context(), loc)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (m *timezoneMiddleware) resolve(ctx context.Context, userID string) *time.Location {
	if userID == "" || m.cfg.LookupTimezone == nil {
		return m.defaultLoc
	}

	m.mu.RLock()
	if tz, ok := m.cache[userID]; ok {
		m.mu.RUnlock()
		return m.loadLoc(tz)
	}
	m.mu.RUnlock()

	tz := m.cfg.LookupTimezone(ctx, userID)
	if tz == "" {
		log.Printf("[tz-mw] no timezone for uid=%s, using default", userID)
		return m.defaultLoc
	}

	m.mu.Lock()
	m.cache[userID] = tz
	m.mu.Unlock()
	return m.loadLoc(tz)
}

func (m *timezoneMiddleware) loadLoc(name string) *time.Location {
	if name == "" {
		return m.defaultLoc
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return m.defaultLoc
	}
	return loc
}

func isTimezoneStaticAsset(path string) bool {
	return strings.HasPrefix(path, "/assets/") || path == "/favicon.ico"
}
