//go:build fiber

package middleware

import (
	"os"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"leapfor.xyz/espyna/internal/application/ports"
	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// AuthenticationMiddleware provides authentication middleware for Fiber
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

// RequireAuth is a Fiber middleware that validates authentication tokens
func (m *AuthenticationMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth if disabled or service unavailable
		if m.authService == nil || !m.authService.IsEnabled() {
			return c.Next()
		}

		// Check for routes that don't require authentication
		if m.isPublicRoute(c.Path()) {
			return c.Next()
		}

		// Check for API key authentication
		if m.isAuthorizedAPIKey(c) {
			return c.Next()
		}

		// Extract token from Authorization header or cookie
		token := m.extractToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid authorization token",
			})
		}

		// Verify the authentication token using proto types
		req := &authpb.ValidateJwtTokenRequest{
			Token:    token,
			Provider: authpb.Provider_PROVIDER_GCP, // Default provider
		}

		resp, err := m.authService.VerifyToken(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Authentication failed",
			})
		}

		if !resp.IsValid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": resp.ErrorMessage,
			})
		}

		// Add user information to Fiber context
		c.Locals("uid", resp.Identity.Id)
		c.Locals("email", resp.Identity.Email)
		c.Locals("identity", resp.Identity)
		if resp.Token != nil && resp.Token.ExpiresAt != nil {
			c.Locals("expires", resp.Token.ExpiresAt.AsTime().Unix())
		}

		return c.Next()
	}
}

// extractToken extracts the token from Authorization header or cookie
func (m *AuthenticationMiddleware) extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// Try token cookie as fallback
	return c.Cookies("token")
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
func (m *AuthenticationMiddleware) isAuthorizedAPIKey(c *fiber.Ctx) bool {
	// Check X-API-Key header
	apiKey := c.Get("X-API-Key")
	if apiKey != "" && apiKey == os.Getenv("X_API_KEY") {
		return true
	}

	// Check X-API-Key-Scheduler header
	schedulerKey := c.Get("X-API-Key-Scheduler")
	if schedulerKey != "" && schedulerKey == os.Getenv("X_API_KEY_SCHEDULER") {
		return true
	}

	return false
}

// GetUserFromFiberContext extracts user information from Fiber context
func GetUserFromFiberContext(c *fiber.Ctx) (uid string, email string, ok bool) {
	uidVal := c.Locals("uid")
	emailVal := c.Locals("email")

	if uidVal == nil {
		return "", "", false
	}

	uid, uidOk := uidVal.(string)
	email, emailOk := emailVal.(string)

	return uid, email, uidOk && emailOk
}

// GetIdentityFromFiberContext extracts the full identity from Fiber context
func GetIdentityFromFiberContext(c *fiber.Ctx) (*authpb.Identity, bool) {
	identityVal := c.Locals("identity")
	if identityVal == nil {
		return nil, false
	}

	identity, ok := identityVal.(*authpb.Identity)
	return identity, ok
}