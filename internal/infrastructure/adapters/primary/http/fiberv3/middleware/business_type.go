//go:build fiber_v3

package middleware

import (
	"os"

	"github.com/gofiber/fiber/v3"
)

// DefaultBusinessType is the fallback business type
const DefaultBusinessType = "education"

// BusinessTypeMiddleware provides business type handling middleware for Fiber v3
type BusinessTypeMiddleware struct {
	defaultBusinessType string
}

// NewBusinessTypeMiddleware creates a new business type middleware instance
func NewBusinessTypeMiddleware(defaultBusinessType string) *BusinessTypeMiddleware {
	// Check environment variable first, like context helpers
	if envBusinessType := os.Getenv("BUSINESS_TYPE"); envBusinessType != "" {
		defaultBusinessType = envBusinessType
	} else if defaultBusinessType == "" {
		defaultBusinessType = DefaultBusinessType
	}

	return &BusinessTypeMiddleware{
		defaultBusinessType: defaultBusinessType,
	}
}

// SetBusinessType is a Fiber v3 middleware that sets the business type in context
func (m *BusinessTypeMiddleware) SetBusinessType() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Try to get business type from various sources
		businessType := m.extractBusinessType(c)
		
		// Set business type in context for downstream handlers
		c.Locals("businessType", businessType)
		
		return c.Next()
	}
}

// extractBusinessType extracts business type from request headers, query params, or uses default
func (m *BusinessTypeMiddleware) extractBusinessType(c fiber.Ctx) string {
	// Check environment variable first (highest priority)
	if businessType := os.Getenv("BUSINESS_TYPE"); businessType != "" {
		return businessType
	}

	// Check X-Leapfor-MockBusinessType header second
	if businessType := c.Get("X-Leapfor-MockBusinessType"); businessType != "" {
		return businessType
	}
	
	// Check business_type query parameter
	if businessType := c.Query("business_type"); businessType != "" {
		return businessType
	}
	
	// Check businessType query parameter (alternative)
	if businessType := c.Query("businessType"); businessType != "" {
		return businessType
	}
	
	// Fall back to default
	if m.defaultBusinessType != "" {
		return m.defaultBusinessType
	}
	
	// Ultimate fallback
	return DefaultBusinessType
}

// GetBusinessTypeFromContext extracts business type from Fiber v3 context
func GetBusinessTypeFromFiberContext(c fiber.Ctx) string {
	if businessType := c.Locals("businessType"); businessType != nil {
		if bt, ok := businessType.(string); ok {
			return bt
		}
	}
	return DefaultBusinessType
}