//go:build fiber_v3

package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

// CSRF creates a Fiber v3-specific CSRF middleware
func CSRF() fiber.Handler {
	return csrf.New(csrf.Config{
		CookieName:        "csrf_",
		CookieSameSite:    "Lax",
		CookieSecure:      false,
		CookieHTTPOnly:    true,
		CookieSessionOnly: true,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token validation failed",
			})
		},
	})
}