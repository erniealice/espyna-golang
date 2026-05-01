//go:build fiber

package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

// Gzip creates a Fiber-specific gzip compression middleware
func Gzip() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelDefault,
	})
}
