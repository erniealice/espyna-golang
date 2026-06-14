//go:build fiber

package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	// envKeyCookieSecureFiber gates the Secure flag on session, CSRF, and
	// rotated-session cookies. Default true (secure-by-default); set
	// COOKIE_SECURE=false for local HTTP dev.
	// Named differently from the vanilla constant to avoid collision when
	// both packages are imported indirectly.
	envKeyCookieSecureFiber = "COOKIE_SECURE"
)

// secureCookiesFiber is the process-wide Secure-flag decision for the fiber
// adapter. Defaults to true so a writer reached before CookieSecure runs
// fails closed to the secure setting.
var secureCookiesFiber = true

// CookieSecureFromEnv resolves the Secure-flag policy from the environment,
// defaulting to true (secure-by-default). Returns false only when
// COOKIE_SECURE is an explicit falsey value: "false", "0", "no", or "off"
// (case-insensitive).
// Mirrors the vanilla CookieSecureFromEnv.
func CookieSecureFromEnv(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(envKeyCookieSecureFiber))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// SetSecureCookies sets the process-wide Secure-flag decision. Call exactly
// once at boot before serving. Mirrors the vanilla SetSecureCookies.
func SetSecureCookies(v bool) bool {
	secureCookiesFiber = v
	return v
}

// SecureCookies reports the current process-wide Secure-flag decision.
func SecureCookies() bool { return secureCookiesFiber }

// CookieSecure returns a Fiber middleware that resolves the COOKIE_SECURE
// environment variable at construction time and calls SetSecureCookies.
// The middleware itself is a pass-through (the Secure-flag policy is
// process-wide, not per-request). Calling CookieSecure() initializes the
// policy as a side effect so downstream cookie writers read the correct
// value from SecureCookies().
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/cookie_secure.go).
func CookieSecure() fiber.Handler {
	SetSecureCookies(CookieSecureFromEnv(os.Getenv))
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
