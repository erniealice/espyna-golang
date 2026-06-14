//go:build fiber

package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// ActionGuardConfig configures the ActionGuard middleware wrapper for Fiber.
//
// The full implementation of the signed _workspace_id form-field verification
// lives in the service-admin middleware package (action_workspace_guard.go).
// This config struct provides the closure interface so the espyna fiber adapter
// can construct the middleware without importing service-admin internals.
//
// Mirrors the vanilla ActionGuardConfig
// (consumer/http/middleware/action_guard.go).
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
	// from the Fiber request context. When nil defaults to reading
	// from the user context via identity.Must.
	WorkspaceIDFromContext func(c *fiber.Ctx) string

	// Handler is a pre-built Fiber middleware function. When set, all other
	// config fields are ignored and this handler is used directly. This
	// allows the caller to pass the output of service-admin's
	// NewActionWorkspaceGuardMiddleware through. When nil, the config
	// fields above are used (future built-in implementation; currently
	// falls back to pass-through when Signer is nil).
	Handler fiber.Handler
}

// ActionGuard returns a Fiber middleware that enforces the signed
// _workspace_id hidden-field invariant on /action/* mutating requests.
// This prevents cross-workspace form submission after URL-driven session
// rotation (red-team X-3 / A-3 / C-3).
//
// When cfg.Handler is set, it is used directly (the caller has already
// constructed the full middleware). When cfg.Signer is nil and no Handler
// is provided, the middleware is a pass-through (action guard disabled).
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/action_guard.go).
//
// TODO: When the full action-guard implementation moves to espyna's fiber
// contrib, it will be constructed here from the config fields.
func ActionGuard(cfg ActionGuardConfig) fiber.Handler {
	// Fast path: pre-built handler provided.
	if cfg.Handler != nil {
		return cfg.Handler
	}

	// No signer: disabled (pass-through).
	if cfg.Signer == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	// Future: when the full action-guard implementation moves to espyna,
	// it will be constructed here from the config fields. For now,
	// callers must set cfg.Handler to delegate to service-admin's
	// middleware.
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
