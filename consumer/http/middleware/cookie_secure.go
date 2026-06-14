package middleware

import (
	"net/http"
	"os"
	"strings"
)

// defaultGetenv is the standard os.Getenv, extracted so tests can override
// via the FromEnv variants.
var defaultGetenv = os.Getenv

const (
	// EnvKeyCookieSecure gates the Secure flag on session, CSRF, and
	// rotated-session cookies. Default true (secure-by-default); set
	// COOKIE_SECURE=false for local HTTP dev.
	EnvKeyCookieSecure = "COOKIE_SECURE"
)

// secureCookies is the process-wide Secure-flag decision. Defaults to true
// so a writer reached before SetSecureCookies fails closed to the secure
// setting.
var secureCookies = true

// CookieSecureFromEnv resolves the Secure-flag policy from the environment,
// defaulting to true (secure-by-default). Returns false only when
// COOKIE_SECURE is an explicit falsey value: "false", "0", "no", or "off"
// (case-insensitive).
func CookieSecureFromEnv(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(EnvKeyCookieSecure))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// SetSecureCookies sets the process-wide Secure-flag decision. Call exactly
// once at boot before serving.
func SetSecureCookies(v bool) bool {
	secureCookies = v
	return v
}

// SecureCookies reports the current process-wide Secure-flag decision.
func SecureCookies() bool { return secureCookies }

// CookieSecure returns a MiddlewareFunc that resolves the COOKIE_SECURE
// environment variable at construction time and calls SetSecureCookies.
// The middleware itself is a pass-through (the Secure-flag policy is
// process-wide, not per-request). Calling CookieSecure() initializes the
// policy as a side effect so downstream cookie writers read the correct
// value from SecureCookies().
func CookieSecure() MiddlewareFunc {
	SetSecureCookies(CookieSecureFromEnv(defaultGetenv))
	return func(next http.Handler) http.Handler {
		return next
	}
}
