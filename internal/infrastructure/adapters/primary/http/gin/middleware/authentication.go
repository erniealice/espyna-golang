//go:build gin

package middleware

import (
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"leapfor.xyz/espyna/internal/application/ports"
	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// AuthenticationMiddleware provides authentication middleware for Gin
type AuthenticationMiddleware struct {
	authService  ports.AuthService
	publicRoutes []string
}

// NewAuthenticationMiddleware creates a new authentication middleware instance
func NewAuthenticationMiddleware(authService ports.AuthService, publicRoutes []string) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		authService:  authService,
		publicRoutes: publicRoutes,
	}
}

// RequireAuth is a Gin middleware that validates authentication tokens
func (m *AuthenticationMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if disabled or service unavailable
		if m.authService == nil || !m.authService.IsEnabled() {
			c.Next()
			return
		}

		// Check for routes that don't require authentication
		if m.isPublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Check for API key authentication
		if m.isAuthorizedAPIKey(c) {
			c.Next()
			return
		}

		// Extract token from Authorization header or cookie
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid authorization token",
			})
			c.Abort()
			return
		}

		// Verify the authentication token using proto types
		req := &authpb.ValidateJwtTokenRequest{
			Token:    token,
			Provider: authpb.Provider_PROVIDER_GCP, // Default provider
		}

		resp, err := m.authService.VerifyToken(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication failed",
			})
			c.Abort()
			return
		}

		if !resp.IsValid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": resp.ErrorMessage,
			})
			c.Abort()
			return
		}

		// Add user information to Gin context
		c.Set("uid", resp.Identity.Id)
		c.Set("email", resp.Identity.Email)
		c.Set("identity", resp.Identity)
		if resp.Token != nil && resp.Token.ExpiresAt != nil {
			c.Set("expires", resp.Token.ExpiresAt.AsTime().Unix())
		}

		c.Next()
	}
}

// extractToken extracts the token from Authorization header or cookie
func (m *AuthenticationMiddleware) extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// Try token cookie as fallback
	if token, err := c.Cookie("token"); err == nil {
		return token
	}

	return ""
}

// isPublicRoute checks if the route is public (no auth required)
func (m *AuthenticationMiddleware) isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/health",
		"/api/ping", // Health check endpoints
	}

	return slices.Contains(publicRoutes, path)
}

// isAuthorizedAPIKey checks for valid API keys
func (m *AuthenticationMiddleware) isAuthorizedAPIKey(c *gin.Context) bool {
	// Check X-API-Key header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" && apiKey == os.Getenv("X_API_KEY") {
		return true
	}

	// Check X-API-Key-Scheduler header
	schedulerKey := c.GetHeader("X-API-Key-Scheduler")
	if schedulerKey != "" && schedulerKey == os.Getenv("X_API_KEY_SCHEDULER") {
		return true
	}

	return false
}

// GetUserFromGinContext extracts user information from Gin context
func GetUserFromGinContext(c *gin.Context) (uid string, email string, ok bool) {
	uidVal, uidExists := c.Get("uid")
	emailVal, _ := c.Get("email")

	if !uidExists {
		return "", "", false
	}

	uid, uidOk := uidVal.(string)
	email, emailOk := emailVal.(string)

	return uid, email, uidOk && emailOk
}

// GetIdentityFromGinContext extracts the full identity from Gin context
func GetIdentityFromGinContext(c *gin.Context) (*authpb.Identity, bool) {
	identityVal, exists := c.Get("identity")
	if !exists {
		return nil, false
	}

	identity, ok := identityVal.(*authpb.Identity)
	return identity, ok
}