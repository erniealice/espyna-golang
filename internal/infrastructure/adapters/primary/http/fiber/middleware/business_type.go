//go:build fiber

package middleware

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
)

// BusinessTypeMiddleware handles business type extraction from HTTP headers for Fiber
func BusinessTypeMiddleware(defaultBusinessType string) fiber.Handler {
	// Check environment variable first, like context helpers
	if envBusinessType := os.Getenv("BUSINESS_TYPE"); envBusinessType != "" {
		defaultBusinessType = envBusinessType
	} else if defaultBusinessType == "" {
		defaultBusinessType = "education" // Fallback default
	}
	
	// Valid business types
	validBusinessTypes := map[string]bool{
		"education":        true,
		"fitness_center":   true,
		"office_leasing":   true,
		"aesthetic_clinic": true,
		"general":          true,
	}
	
	return func(c *fiber.Ctx) error {
		// Check environment variable first, then header, then default
		businessType := os.Getenv("BUSINESS_TYPE")
		if businessType == "" {
			businessType = c.Get("X-Leapfor-MockBusinessType")
		}
		if businessType == "" {
			businessType = defaultBusinessType
		}
		
		// Validate business type
		if !validBusinessTypes[businessType] {
			// Invalid business type - fall back to default
			businessType = defaultBusinessType
		}
		
		// Set business type in context
		ctx := context.WithValue(c.Context(), "businessType", businessType)
		c.SetUserContext(ctx)
		
		// Continue with request processing
		return c.Next()
	}
}