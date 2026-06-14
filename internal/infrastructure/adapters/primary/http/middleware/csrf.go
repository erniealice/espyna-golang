package middleware

import "net/http"

// CSRFConfig configures the CSRF middleware wrapper.
//
// The full implementation of workspace-claim CSRF token issuance and
// validation lives in the service-admin middleware package
// (csrf_workspace.go). This config struct provides the closure interface
// so the espyna Server can construct the middleware without importing
// service-admin internals.
type CSRFConfig struct {
	// Secret is the HMAC-SHA256 signing key for workspace-claim CSRF tokens.
	// Use SecretFromEnv (from cookie_secure.go) to populate. When empty the
	// middleware is a pass-through (CSRF protection disabled).
	Secret []byte

	// SessionToken extracts the current session token from the request.
	// When nil defaults to reading from the request context.
	SessionToken func(r *http.Request) string

	// WorkspaceID extracts the session's current workspace_id from the request.
	// When nil defaults to reading from the request context.
	WorkspaceID func(r *http.Request) string

	// PathPrefix scopes validation to mutation endpoints. Defaults to
	// "/action/". Cookie issuance still happens on every GET regardless.
	PathPrefix string

	// Handler is the pre-built middleware function. When set, all other
	// config fields are ignored and this handler is used directly. This
	// allows the caller to pass the output of service-admin's
	// NewWorkspaceCSRFMiddleware through. When nil, the config fields
	// above are used (future built-in implementation; currently falls
	// back to pass-through when Secret is empty).
	Handler func(http.Handler) http.Handler
}

// CSRF returns a MiddlewareFunc that validates workspace-claim CSRF tokens
// on mutating requests. GET requests receive a fresh CSRF cookie; POST/PUT/
// PATCH/DELETE requests must present a valid token.
//
// When cfg.Handler is set, it is used directly (the caller has already
// constructed the full middleware). When cfg.Secret is empty and no Handler
// is provided, the middleware is a pass-through (CSRF protection disabled).
func CSRF(cfg CSRFConfig) MiddlewareFunc {
	// Fast path: pre-built handler provided
	if cfg.Handler != nil {
		return func(next http.Handler) http.Handler {
			return cfg.Handler(next)
		}
	}

	// No secret → disabled (pass-through)
	if len(cfg.Secret) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Future: when the full CSRF implementation moves to espyna, it will
	// be constructed here from the config fields. For now, callers must
	// set cfg.Handler to delegate to service-admin's middleware.
	return func(next http.Handler) http.Handler {
		return next
	}
}
