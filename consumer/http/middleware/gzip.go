package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// Gzip returns a MiddlewareFunc that compresses responses for clients that
// accept gzip encoding. Pre-compressed formats (images, fonts, archives) are
// excluded automatically.
func Gzip() MiddlewareFunc { return impl.Gzip() }
