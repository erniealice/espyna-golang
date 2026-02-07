//go:build vanilla

package middleware

import (
	"context"
	"net/http"
	"os"
)

// BusinessTypeMiddleware handles business type extraction and context setting
type BusinessTypeMiddleware struct {
	defaultBusinessType string
}

// NewBusinessTypeMiddleware creates a new business type middleware
func NewBusinessTypeMiddleware(defaultBusinessType string) *BusinessTypeMiddleware {
	// Check environment variable first, like context helpers
	if envBusinessType := os.Getenv("BUSINESS_TYPE"); envBusinessType != "" {
		defaultBusinessType = envBusinessType
	} else if defaultBusinessType == "" {
		defaultBusinessType = "education" // Fallback default
	}

	return &BusinessTypeMiddleware{
		defaultBusinessType: defaultBusinessType,
	}
}

// SetBusinessType middleware extracts business type from HTTP headers and sets it in context
func (m *BusinessTypeMiddleware) SetBusinessType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check environment variable first, then header, then default
		businessType := os.Getenv("BUSINESS_TYPE")
		if businessType == "" {
			businessType = r.Header.Get("X-Leapfor-MockBusinessType")
		}
		if businessType == "" {
			businessType = m.defaultBusinessType
		}
		
		// Validate business type (optional - you can expand this list)
		validBusinessTypes := map[string]bool{
			"education":        true,
			"fitness_center":   true,
			"office_leasing":   true,
			"aesthetic_clinic": true,
			"general":          true,
		}
		
		if !validBusinessTypes[businessType] {
			// Invalid business type - fall back to default
			businessType = m.defaultBusinessType
		}
		
		// Set business type in context
		ctx := context.WithValue(r.Context(), "businessType", businessType)
		
		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}