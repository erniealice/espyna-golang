package middleware

import "net/http"

// CSRF returns a MiddlewareFunc that validates workspace-claim CSRF tokens
// on mutating requests. GET requests receive a fresh CSRF cookie; POST/PUT/
// PATCH/DELETE requests must present a valid token.
//
// Stub: the full implementation will be wired when the Server API lands.
// The middleware will accept a WorkspaceCSRFConfig carrying the HMAC secret
// and session/workspace context readers.
func CSRF() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		// Pass-through until the Server API provides CSRF config.
		return next
	}
}
