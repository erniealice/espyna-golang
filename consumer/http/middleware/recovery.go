package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// Recovery returns a MiddlewareFunc that recovers from panics in downstream
// handlers and responds with a 500 Internal Server Error.
func Recovery() MiddlewareFunc { return impl.Recovery() }
