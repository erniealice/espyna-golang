// Package middleware is the public API surface for espyna HTTP middleware.
// Each function returns a MiddlewareFunc that wraps an http.Handler.
// Implementations live in internal/infrastructure/adapters/primary/http/middleware.
package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// MiddlewareFunc is a function that wraps an http.Handler with middleware
// behaviour. This is the standard Go middleware signature.
type MiddlewareFunc = impl.MiddlewareFunc
