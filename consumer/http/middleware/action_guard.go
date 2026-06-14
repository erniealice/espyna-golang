package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// ActionGuardConfig configures the ActionGuard middleware wrapper.
type ActionGuardConfig = impl.ActionGuardConfig

// ActionGuard returns a MiddlewareFunc that enforces the signed
// _workspace_id hidden-field invariant on /action/* mutating requests.
func ActionGuard(cfg ActionGuardConfig) MiddlewareFunc { return impl.ActionGuard(cfg) }
