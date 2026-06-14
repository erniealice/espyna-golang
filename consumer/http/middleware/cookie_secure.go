package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

const (
	// EnvKeyCookieSecure gates the Secure flag on session, CSRF, and
	// rotated-session cookies. Default true (secure-by-default); set
	// COOKIE_SECURE=false for local HTTP dev.
	EnvKeyCookieSecure = impl.EnvKeyCookieSecure
)

// CookieSecureFromEnv resolves the Secure-flag policy from the environment.
func CookieSecureFromEnv(getenv func(string) string) bool { return impl.CookieSecureFromEnv(getenv) }

// SetSecureCookies sets the process-wide Secure-flag decision.
func SetSecureCookies(v bool) bool { return impl.SetSecureCookies(v) }

// SecureCookies reports the current process-wide Secure-flag decision.
func SecureCookies() bool { return impl.SecureCookies() }

// CookieSecure returns a MiddlewareFunc that resolves the COOKIE_SECURE
// environment variable at construction time and calls SetSecureCookies.
func CookieSecure() MiddlewareFunc { return impl.CookieSecure() }
