package middleware

import (
	"net/http"

	impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"
)

// SessionHandler is the interface satisfied by both consumer.SessionMiddleware
// and consumer.MockSessionMiddleware.
type SessionHandler = impl.SessionHandler

// MockSessionHandler wraps a consumer.MockSessionMiddleware.Handle method
// into a SessionHandler.
func MockSessionHandler(handle func(http.Handler) http.Handler) SessionHandler {
	return impl.MockSessionHandler(handle)
}

// Session returns a MiddlewareFunc that delegates to the given session
// handler for session validation.
func Session(handler SessionHandler) MiddlewareFunc { return impl.Session(handler) }
