package middleware

import "net/http"

// ActionGuard returns a MiddlewareFunc that enforces the signed
// _workspace_id hidden-field invariant on /action/* mutating requests.
// This prevents cross-workspace form submission after URL-driven session
// rotation (red-team X-3 / A-3 / C-3).
//
// Stub: the full implementation will be wired when the Server API lands.
// The middleware will accept an ActionGuardConfig carrying the HMAC signer
// and session workspace ID reader.
func ActionGuard() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		// Pass-through until the Server API provides action-guard config.
		return next
	}
}
