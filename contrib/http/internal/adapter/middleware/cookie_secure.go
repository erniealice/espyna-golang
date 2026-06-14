//go:build http

// cookie_secure.go
//
// Process-wide Secure-flag policy for cookies written by middleware in this
// package (the workspace-claim CSRF cookie and the URL-rotation strict session
// cookie). Secure-by-default: ON unless COOKIE_SECURE is explicitly set to a
// falsey value. Every TLS deployment gets Secure cookies with zero config;
// local http dev opts out with COOKIE_SECURE=false.
package middleware

import "strings"

// EnvKeyCookieSecure gates the Secure flag on the session, CSRF, and rotated-
// session cookies. Default true; set COOKIE_SECURE=false (or 0/no/off) for
// local http dev.
const EnvKeyCookieSecure = "COOKIE_SECURE"

// secureCookies is the process-wide Secure-flag decision. It is resolved once
// at boot -- composition/container.go calls SetSecureCookies(CookieSecureFromEnv(...))
// before the server starts serving -- and never mutated afterwards, so the
// concurrent reads during request handling need no synchronization. Defaults to
// true so a writer reached before SetSecureCookies still fails closed to the
// secure setting.
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
// at boot (the composition root) before serving. Returns the value for
// convenient assignment to the espyna SessionMiddleware.CookieSecure field.
func SetSecureCookies(v bool) bool {
	secureCookies = v
	return v
}

// SecureCookies reports the current process-wide Secure-flag decision. Exported
// for tests and for any future cookie writer that needs the same policy.
func SecureCookies() bool { return secureCookies }
