package middleware

import "net/http"

// Session returns a MiddlewareFunc that manages authenticated sessions.
//
// Stub: the full implementation will be wired when the Server API lands.
// The middleware will validate session cookies, inject session identity
// (user_id, workspace_id, token) into the request context, and handle
// session expiry / renewal.
func Session() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		// Pass-through until the Server API provides session config.
		return next
	}
}
