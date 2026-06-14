//go:build gin

// cookie_secure.go — process-wide Secure-flag policy for cookies.
//
// Mirrors the vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/cookie_secure.go).
//
// Unlike HSTS (opt-IN, OFF by default), cookie Secure is SECURE-BY-DEFAULT:
// ON unless COOKIE_SECURE is explicitly set to a falsey value. Every TLS
// deployment gets Secure cookies with zero config; local http dev opts out
// with COOKIE_SECURE=false.
package middleware

import "strings"

// EnvKeyCookieSecure gates the Secure flag on session, CSRF, and rotated-
// session cookies. Default true; set COOKIE_SECURE=false (or 0/no/off) for
// local http dev.
const EnvKeyCookieSecure = "COOKIE_SECURE"

// secureCookies is the process-wide Secure-flag decision. Resolved once at
// boot and never mutated afterwards.
var secureCookies = true

// CookieSecureFromEnv resolves the Secure-flag policy from the environment,
// defaulting to true (secure-by-default). Returns false only when COOKIE_SECURE
// is an explicit falsey value: "false", "0", "no", or "off" (case-insensitive).
func CookieSecureFromEnv(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(EnvKeyCookieSecure))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// SetSecureCookies sets the process-wide Secure-flag decision. Call exactly once
// at boot before serving. Returns the value for convenient assignment.
func SetSecureCookies(v bool) bool {
	secureCookies = v
	return v
}

// SecureCookies reports the current process-wide Secure-flag decision.
func SecureCookies() bool { return secureCookies }
