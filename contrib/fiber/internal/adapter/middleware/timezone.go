//go:build fiber

package middleware

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TimezoneConfig configures the Timezone middleware for Fiber.
// Mirrors the vanilla TimezoneConfig (consumer/http/middleware/timezone.go).
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

// fiberTimezoneMiddleware resolves the operator's preferred display timezone
// once per request and stashes a *time.Location on the request user context.
type fiberTimezoneMiddleware struct {
	cfg        TimezoneConfig
	defaultLoc *time.Location
	mu         sync.RWMutex
	cache      map[string]string
}

// Timezone returns a Fiber middleware that resolves the authenticated user's
// timezone preference and stores the resulting *time.Location on the request
// user context. Non-authenticated requests and static asset paths pass through
// without a timezone lookup.
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/timezone.go): same caching, same static-asset
// skip, same default "Asia/Manila".
func Timezone(cfg TimezoneConfig) fiber.Handler {
	if cfg.DefaultTimezone == "" {
		cfg.DefaultTimezone = "Asia/Manila"
	}
	defaultLoc, err := time.LoadLocation(cfg.DefaultTimezone)
	if err != nil {
		defaultLoc = time.UTC
	}
	m := &fiberTimezoneMiddleware{
		cfg:        cfg,
		defaultLoc: defaultLoc,
		cache:      make(map[string]string),
	}
	return m.handle
}

func (m *fiberTimezoneMiddleware) handle(c *fiber.Ctx) error {
	if isFiberTimezoneStaticAsset(c.Path()) {
		return c.Next()
	}

	var userID string
	if m.cfg.GetUserID != nil {
		userID = m.cfg.GetUserID(c.UserContext())
	}

	loc := m.resolve(c.UserContext(), userID)
	if m.cfg.StoreLocation != nil {
		ctx := m.cfg.StoreLocation(c.UserContext(), loc)
		c.SetUserContext(ctx)
	}

	return c.Next()
}

func (m *fiberTimezoneMiddleware) resolve(ctx context.Context, userID string) *time.Location {
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

func (m *fiberTimezoneMiddleware) loadLoc(name string) *time.Location {
	if name == "" {
		return m.defaultLoc
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return m.defaultLoc
	}
	return loc
}

func isFiberTimezoneStaticAsset(path string) bool {
	return strings.HasPrefix(path, "/assets/") || path == "/favicon.ico"
}
