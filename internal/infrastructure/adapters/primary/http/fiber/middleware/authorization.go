//go:build fiber

package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// AuthorizationMiddleware provides authorization middleware for Fiber requests
type AuthorizationMiddleware struct {
	authorizationService ports.AuthorizationService
}

// NewAuthorizationMiddleware creates a new authorization middleware instance
func NewAuthorizationMiddleware(authorizationService ports.AuthorizationService) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authorizationService: authorizationService,
	}
}

// RequirePermission creates middleware that requires a specific permission
func (m *AuthorizationMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromFiberContext(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Check permission
		authorized, err := m.authorizationService.HasPermission(c.Context(), userID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Authorization check failed",
			})
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequireWorkspacePermission creates middleware that requires a workspace-specific permission
func (m *AuthorizationMiddleware) RequireWorkspacePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromFiberContext(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Extract workspace ID from context or URL parameters
		workspaceID := GetWorkspaceFromFiberContext(c)
		if workspaceID == "" {
			workspaceID = c.Params("workspace_id")
		}

		if workspaceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Workspace context required",
			})
		}

		// Check workspace-specific permission
		authorized, err := m.authorizationService.HasPermissionInWorkspace(c.Context(), userID, workspaceID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Authorization check failed",
			})
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient workspace permissions",
			})
		}

		return c.Next()
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles
func (m *AuthorizationMiddleware) RequireAnyRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromFiberContext(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Get user roles
		userRoles, err := m.authorizationService.GetUserRoles(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get user roles",
			})
		}

		// Check if user has any of the required roles
		authorized := m.hasAnyRole(userRoles, roles)
		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient role permissions",
			})
		}

		return c.Next()
	}
}

// RequireWorkspaceRole creates middleware that requires a specific role within a workspace
func (m *AuthorizationMiddleware) RequireWorkspaceRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromFiberContext(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// Extract workspace ID from context or URL parameters
		workspaceID := GetWorkspaceFromFiberContext(c)
		if workspaceID == "" {
			workspaceID = c.Params("workspace_id")
		}

		if workspaceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Workspace context required",
			})
		}

		// Get user roles in workspace
		workspaceRoles, err := m.authorizationService.GetUserRolesInWorkspace(c.Context(), userID, workspaceID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get workspace roles",
			})
		}

		// Check if user has required role in workspace
		authorized := m.hasRole(workspaceRoles, role)
		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient workspace role",
			})
		}

		return c.Next()
	}
}

// Helper functions

// GetUserIDFromFiberContext extracts user ID from the Fiber context
// This should be set by the authentication middleware
func GetUserIDFromFiberContext(c *fiber.Ctx) string {
	// Check for "uid" (set by authentication middleware)
	uid := c.Locals("uid")
	if uid != nil {
		if uidStr, ok := uid.(string); ok {
			return uidStr
		}
	}

	// Fallback to "user_id" if available
	userID := c.Locals("user_id")
	if userID != nil {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}

	return ""
}

// GetWorkspaceFromFiberContext extracts workspace ID from the Fiber context
func GetWorkspaceFromFiberContext(c *fiber.Ctx) string {
	workspaceID := c.Locals("workspace_id")
	if workspaceID != nil {
		if workspaceIDStr, ok := workspaceID.(string); ok {
			return workspaceIDStr
		}
	}
	return ""
}

// hasAnyRole checks if user has any of the required roles
func (m *AuthorizationMiddleware) hasAnyRole(userRoles []string, requiredRoles []string) bool {
	for _, userRole := range userRoles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

// hasRole checks if user has a specific role
func (m *AuthorizationMiddleware) hasRole(userRoles []string, requiredRole string) bool {
	for _, userRole := range userRoles {
		if userRole == requiredRole {
			return true
		}
	}
	return false
}
