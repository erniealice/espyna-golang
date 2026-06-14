package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// Logger returns a MiddlewareFunc that logs every HTTP request with method,
// path, status code, and duration.
func Logger() MiddlewareFunc { return impl.Logger() }
