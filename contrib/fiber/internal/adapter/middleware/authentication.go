//go:build fiber

package middleware

import (
	"os"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/erniealice/espyna-golang/ports"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// AuthenticationMiddleware provides authentication middleware for Fiber requests.
//
// Security semantics mirror the vanilla net/http reference implementation
// (contrib/http/internal/adapter/middleware/authentication.go): same fail-open
// when the auth service is disabled, same public-route allowlist, same API-key
// bypass, same Bearer/cookie token extraction, same proto VerifyToken contract,
// and the same failure modes (401 missing/invalid token, 500 verify error,
// 401 invalid result). Only the framework surface (*fiber.Ctx) differs.
type AuthenticationMiddleware struct {
	authService ports.AuthService
}

// NewAuthenticationMiddleware creates a new authentication middleware instance.
func NewAuthenticationMiddleware(authService ports.AuthService) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		authService: authService,
	}
}

// RequireAuth is a Fiber middleware that validates authentication tokens.
func (m *AuthenticationMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth if disabled or service unavailable.
		if m.authService == nil || !m.authService.IsEnabled() {
			return c.Next()
		}

		// Check for routes that don't require authentication.
		if m.isPublicRoute(c.Path()) {
			return c.Next()
		}

		// Check for API key authentication.
		if m.isAuthorizedAPIKey(c) {
			return c.Next()
		}

		// Extract token from Authorization header or cookie.
		token := m.extractToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid authorization token",
			})
		}

		// Verify the authentication token using proto types.
		req := &authpb.ValidateJwtTokenRequest{
			Token:    token,
			Provider: authpb.Provider_PROVIDER_GCP, // Default provider, could be configured
		}

		resp, err := m.authService.VerifyToken(c.UserContext(), req)
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

		// Add user information to the request user context.
		//
		// SECURITY: Do NOT write identity.RequestIdentity here. This JWT-based
		// auth middleware only knows UserID/Email — it has no workspace context.
		// Writing a RequestIdentity with empty WorkspaceID would cause
		// identity.Must(ctx).WorkspaceID to return "" instead of panicking,
		// which disables tenant filtering on fail-open SQL predicates.
		// The session middleware resolves the full identity and writes
		// RequestIdentity with workspace context populated.
		ctx := contextWithValue(c.UserContext(), ctxKeyIdentity, resp.Identity)
		if resp.Token != nil && resp.Token.ExpiresAt != nil {
			ctx = contextWithValue(ctx, ctxKeyExpires, resp.Token.ExpiresAt.AsTime().Unix())
		}
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// extractToken extracts the token from the Authorization header or cookie.
func (m *AuthenticationMiddleware) extractToken(c *fiber.Ctx) string {
	// Try Authorization header first.
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// Try token cookie as fallback.
	if token := c.Cookies("token"); token != "" {
		return token
	}

	return ""
}

// isPublicRoute checks if the route is public (no auth required).
func (m *AuthenticationMiddleware) isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/health",
		"/api/ping", // Health check endpoints
	}

	return slices.Contains(publicRoutes, path)
}

// isAuthorizedAPIKey checks for valid API keys.
func (m *AuthenticationMiddleware) isAuthorizedAPIKey(c *fiber.Ctx) bool {
	// Check X-API-Key header.
	apiKey := c.Get("X-API-Key")
	if apiKey != "" && apiKey == os.Getenv("X_API_KEY") {
		return true
	}

	// Check X-API-Key-Scheduler header.
	schedulerKey := c.Get("X-API-Key-Scheduler")
	if schedulerKey != "" && schedulerKey == os.Getenv("X_API_KEY_SCHEDULER") {
		return true
	}

	return false
}
