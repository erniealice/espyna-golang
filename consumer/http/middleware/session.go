package middleware

import "net/http"

// SessionHandler is the interface satisfied by both consumer.SessionMiddleware
// (password provider) and a mock-session adapter. It is the declarative Session
// slot the Preset carries; the structurally-identical contrib SessionHandler
// interface accepts it directly via the method set (the chain assembler bridges
// the two). Keeping this an interface (NOT a re-export alias of any impl type)
// is what lets the agnostic surface stay framework-free while remaining
// assignable to every provider's Session impl.
type SessionHandler interface {
	// Handler wraps the given handler with session validation. On each request
	// it validates the session cookie, injects session identity (user_id,
	// workspace_id, token) into the request context, and handles session
	// expiry / renewal.
	Handler(next http.Handler) http.Handler
}

// mockSessionAdapter adapts a Handle-style closure (consumer.MockSessionMiddleware
// exposes Handle, not Handler) to the SessionHandler interface.
type mockSessionAdapter struct {
	handle func(http.Handler) http.Handler
}

func (a *mockSessionAdapter) Handler(next http.Handler) http.Handler {
	return a.handle(next)
}

// MockSessionHandler wraps a consumer.MockSessionMiddleware.Handle method into a
// SessionHandler. MockSessionMiddleware exposes Handle (not Handler), so this
// adapter bridges the naming gap.
//
// Usage:
//
//	mockMw := consumer.NewMockSessionMiddleware(useCases, ...)
//	preset.session = middleware.MockSessionHandler(mockMw.Handle)
func MockSessionHandler(handle func(http.Handler) http.Handler) SessionHandler {
	return &mockSessionAdapter{handle: handle}
}
