//go:build fiber

package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
)

// CSRF creates a Fiber-specific CSRF middleware using the official implementation
// Following Gofiber documentation: https://docs.gofiber.io/api/middleware/csrf/
func CSRF() fiber.Handler {
	return csrf.New(csrf.Config{
		// Token lookup configuration - supports multiple sources
		KeyLookup: "header:X-Csrf-Token,form:_token,query:token",
		
		// Cookie configuration
		CookieName:     "csrf_",
		CookieSameSite: "Lax",
		CookieSecure:   false, // Set to true in production with HTTPS
		CookieHTTPOnly: true,
		CookiePath:     "/",
		
		// Security settings
		Expiration: 1 * time.Hour,
		SingleUseToken: false, // Set to true for single-use tokens
		
		// Custom error handler with consistent API response format
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "CSRF token validation failed",
				"message": "Cross-Site Request Forgery token is missing or invalid",
			})
		},
		
		// Custom extractor function (optional)
		Extractor: func(c *fiber.Ctx) (string, error) {
			// Try header first
			token := c.Get("X-Csrf-Token")
			if token != "" {
				return token, nil
			}
			
			// Try form field
			token = c.FormValue("_token")
			if token != "" {
				return token, nil
			}
			
			// Try query parameter
			token = c.Query("token")
			if token != "" {
				return token, nil
			}
			
			return "", csrf.ErrTokenNotFound
		},
	})
}