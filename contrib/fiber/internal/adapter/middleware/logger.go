//go:build fiber

package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Logger returns a Fiber middleware that logs every request with method, path,
// status code, and duration. Wraps fiber's built-in logger middleware with a
// format that matches the vanilla net/http reference implementation
// (consumer/http/middleware/logger.go).
func Logger() fiber.Handler {
	return logger.New(logger.Config{
		Format:     "${method} ${path} ${status} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	})
}
