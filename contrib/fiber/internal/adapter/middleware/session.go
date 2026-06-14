//go:build fiber

package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// FiberSessionHandler is the interface satisfied by session middleware
// implementations in the Fiber adapter layer. The handler wraps a Fiber
// handler with session validation. On each request it validates the session
// cookie, injects session identity (user_id, workspace_id, token) into the
// request user context, and handles session expiry / renewal.
//
// Mirrors the vanilla SessionHandler interface
// (consumer/http/middleware/session.go) adapted for Fiber's handler signature.
type FiberSessionHandler interface {
	// Handler wraps the given handler with session validation.
	Handler(c *fiber.Ctx) error
}

// Session returns a Fiber middleware that delegates to the given session
// handler for session validation. The handler is typically either:
//
//   - A concrete session middleware implementation (password provider)
//   - A mock session handler (mock provider for testing)
//
// When handler is nil, the middleware is a pass-through (safe for boot-time
// stub configurations where no auth provider is configured).
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/session.go).
//
// TODO: Wire to concrete fiber session handler implementations once the
// session adapter is ported to the fiber contrib layer.
func Session(handler FiberSessionHandler) fiber.Handler {
	if handler == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}
	return func(c *fiber.Ctx) error {
		return handler.Handler(c)
	}
}
