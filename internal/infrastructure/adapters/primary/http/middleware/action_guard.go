package middleware

import "net/http"

// ActionGuardConfig configures the ActionGuard middleware wrapper.
//
// The full implementation of the signed _workspace_id form-field verification
// lives in the service-admin middleware package (action_workspace_guard.go).
// This config struct provides the closure interface so the espyna Server can
// construct the middleware without importing service-admin internals.
type ActionGuardConfig struct {
	// Signer provides HMAC sign/verify operations for the _workspace_id
	// and _workspace_id_sig form fields. When nil the middleware is a
	// pass-through (action guard disabled).
	//
	// In the current architecture, the signer is a *WorkspaceFormSigner
	// from the service-admin middleware package. The interface{} type
	// avoids importing that package; the caller is responsible for
	// providing a signer that satisfies the contract.
	Signer interface{}

	// WorkspaceIDFromContext extracts the session's current workspace_id
	// from the request context. When nil defaults to reading
	// consumer.GetWorkspaceIDFromContext.
	WorkspaceIDFromContext func(r *http.Request) string

	// Handler is the pre-built middleware function. When set, all other
	// config fields are ignored and this handler is used directly. This
	// allows the caller to pass the output of service-admin's
	// NewActionWorkspaceGuardMiddleware through. When nil, the config
	// fields above are used (future built-in implementation; currently
	// falls back to pass-through when Signer is nil).
	Handler func(http.Handler) http.Handler
}

// ActionGuard returns a MiddlewareFunc that enforces the signed
// _workspace_id hidden-field invariant on /action/* mutating requests.
// This prevents cross-workspace form submission after URL-driven session
// rotation (red-team X-3 / A-3 / C-3).
//
// When cfg.Handler is set, it is used directly (the caller has already
// constructed the full middleware). When cfg.Signer is nil and no Handler
// is provided, the middleware is a pass-through (action guard disabled).
func ActionGuard(cfg ActionGuardConfig) MiddlewareFunc {
	// Fast path: pre-built handler provided
	if cfg.Handler != nil {
		return func(next http.Handler) http.Handler {
			return cfg.Handler(next)
		}
	}

	// No signer → disabled (pass-through)
	if cfg.Signer == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Future: when the full action-guard implementation moves to espyna,
	// it will be constructed here from the config fields. For now,
	// callers must set cfg.Handler to delegate to service-admin's
	// middleware.
	return func(next http.Handler) http.Handler {
		return next
	}
}
