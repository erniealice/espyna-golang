package middleware

import "strings"

// EnvKeyCookieSecure gates the Secure flag on session, CSRF, and
// rotated-session cookies. Default true (secure-by-default); set
// COOKIE_SECURE=false for local HTTP dev.
const EnvKeyCookieSecure = "COOKIE_SECURE"

// secureCookies is the agnostic-surface mirror of the process-wide Secure-flag
// decision. NOTE: the LIVE per-request cookie writers read the CONTRIB impl's
// own secureCookies (set by provider.BuildChain via cmw.SetSecureCookies). This
// surface copy exists only so callers that resolve the policy at boot (server.go
// builds the Preset's CookieSecure slot from CookieSecureFromEnv) have a stable
// agnostic API. Defaults true so a read before any Set fails closed to secure.
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

// SetSecureCookies records the process-wide Secure-flag decision on the agnostic
// surface. The authoritative per-request decision is the contrib impl's, set by
// the chain assembler — see secureCookies above.
func SetSecureCookies(v bool) bool {
	secureCookies = v
	return v
}

// SecureCookies reports the agnostic-surface Secure-flag decision.
func SecureCookies() bool { return secureCookies }
