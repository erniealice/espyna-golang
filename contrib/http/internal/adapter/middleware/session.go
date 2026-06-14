//go:build http

// session.go
//
// Session middleware wrapper for the net/http server provider. Delegates to the
// consumer.SessionMiddleware (password provider) or consumer.MockSessionMiddleware
// (mock provider) for session validation, cookie management, and identity injection.
//
// The session middleware validates session cookies on protected routes. If no valid
// session exists, it redirects to the login URL. On valid session, it injects the
// full session identity (user_id, workspace_id, workspace_user_id, session_token)
// into the request context.
package middleware

import "net/http"

// SessionHandler is the interface satisfied by both consumer.SessionMiddleware
// and consumer.MockSessionMiddleware. The consumer types live in
// github.com/erniealice/espyna-golang/consumer -- this interface lets the
// middleware wrapper accept either without importing the concrete types.
type SessionHandler interface {
	// Handler wraps the given handler with session validation. On each
	// request it validates the session cookie, injects session identity
	// (user_id, workspace_id, token) into the request context, and handles
	// session expiry / renewal.
	Handler(next http.Handler) http.Handler
}

// mockSessionAdapter adapts consumer.MockSessionMiddleware (which exposes
// Handle, not Handler) to the SessionHandler interface.
type mockSessionAdapter struct {
	handle func(http.Handler) http.Handler
}

func (a *mockSessionAdapter) Handler(next http.Handler) http.Handler {
	return a.handle(next)
}

// MockSessionHandler wraps a consumer.MockSessionMiddleware.Handle method
// into a SessionHandler. MockSessionMiddleware exposes Handle (not Handler),
// so this adapter bridges the naming gap.
//
// Usage:
//
//	mockMw := consumer.NewMockSessionMiddleware(useCases, ...)
//	sessionMw := middleware.Session(middleware.MockSessionHandler(mockMw.Handle))
func MockSessionHandler(handle func(http.Handler) http.Handler) SessionHandler {
	return &mockSessionAdapter{handle: handle}
}

// Session returns a middleware func that delegates to the given session
// handler for session validation. The handler is typically either:
//
//   - consumer.SessionMiddleware  (password provider)
//   - MockSessionHandler(mockMw.Handle) (mock provider)
//
// When handler is nil, the middleware is a pass-through (safe for boot-time
// stub configurations where no auth provider is configured).
//
// Example:
//
//	// Password provider:
//	sessionMw := consumer.NewSessionMiddleware(authAdapter)
//	server.WithMiddleware(middleware.Session(sessionMw))
//
//	// Mock provider:
//	mockMw := consumer.NewMockSessionMiddleware(useCases, ...)
//	server.WithMiddleware(middleware.Session(middleware.MockSessionHandler(mockMw.Handle)))
func Session(handler SessionHandler) func(http.Handler) http.Handler {
	if handler == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return handler.Handler(next)
	}
}
