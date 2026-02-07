//go:build gin

package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// BusinessTypeMiddleware handles business type extraction from HTTP headers
func BusinessTypeMiddleware(defaultBusinessType string) gin.HandlerFunc {
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
		"business":         true,
	}
	
	return gin.HandlerFunc(func(c *gin.Context) {
		// Check environment variable first, then header, then default
		businessType := os.Getenv("BUSINESS_TYPE")
		if businessType == "" {
			businessType = c.GetHeader("X-Leapfor-MockBusinessType")
		}
		if businessType == "" {
			businessType = defaultBusinessType
		}
		
		// Validate business type
		if !validBusinessTypes[businessType] {
			// Invalid business type - fall back to default
			businessType = defaultBusinessType
		}
		
		// Set business type in Gin context (which uses Go context internally)
		c.Set("businessType", businessType)
		
		// Continue with request processing
		c.Next()
	})
}