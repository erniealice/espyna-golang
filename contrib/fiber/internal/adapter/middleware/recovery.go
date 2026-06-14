//go:build fiber

package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Recovery returns a Fiber middleware that recovers from panics in downstream
// handlers and responds with a 500 Internal Server Error. Wraps fiber's
// built-in recover middleware with a stack-trace logger that mirrors the
// vanilla net/http reference implementation
// (consumer/http/middleware/recovery.go).
func Recovery() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Printf("Panic recovered on %s %s: %v\n%s",
				c.Method(), c.Path(), e, debug.Stack())
		},
	})
}
