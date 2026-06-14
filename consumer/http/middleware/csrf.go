package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// CSRFConfig configures the CSRF middleware wrapper.
type CSRFConfig = impl.CSRFConfig

// CSRF returns a MiddlewareFunc that validates workspace-claim CSRF tokens
// on mutating requests.
func CSRF(cfg CSRFConfig) MiddlewareFunc { return impl.CSRF(cfg) }
