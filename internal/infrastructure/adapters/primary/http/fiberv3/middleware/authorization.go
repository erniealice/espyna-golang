//go:build fiber_v3

package middleware

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"leapfor.xyz/espyna/internal/application/ports"
)

// AuthorizationMiddleware provides authorization middleware for Fiber v3
type AuthorizationMiddleware struct {
	authorizationService ports.AuthorizationService
}

// NewAuthorizationMiddleware creates a new authorization middleware instance
func NewAuthorizationMiddleware(authorizationService ports.AuthorizationService) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authorizationService: authorizationService,
	}
}

// RequirePermission creates a Fiber v3 middleware that validates permissions
func (m *AuthorizationMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Get identity from context (contains user ID)
		identity, ok := GetIdentityFromFiberContext(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No authentication found",
			})
		}

		// Check permission
		ctx := context.Background()
		hasPermission, err := m.authorizationService.HasPermission(ctx, identity.Id, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Authorization check failed",
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequireWorkspacePermission creates a Fiber v3 middleware that validates workspace permissions
func (m *AuthorizationMiddleware) RequireWorkspacePermission(permission string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Get identity from context (contains user ID)
		identity, ok := GetIdentityFromFiberContext(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No authentication found",
			})
		}

		// Extract workspace ID from path parameters
		workspaceID := c.Params("workspaceId")
		if workspaceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Workspace ID is required",
			})
		}

		// Check workspace permission
		ctx := context.Background()
		hasPermission, err := m.authorizationService.HasPermissionInWorkspace(ctx, identity.Id, workspaceID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Authorization check failed",
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient workspace permissions",
			})
		}

		return c.Next()
	}
}