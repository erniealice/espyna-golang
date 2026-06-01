//go:build fiber

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/erniealice/espyna-golang/ports"
)

// AuthorizationMiddleware provides authorization middleware for Fiber requests.
//
// Security semantics mirror the vanilla net/http reference implementation
// (contrib/http/internal/adapter/middleware/authorization.go) exactly: same
// fail-open when the authorizer is disabled, same 401 when the user is not
// authenticated, same 400 when workspace context is required but absent, same
// 403 on insufficient permission/role, and the same identical checks against
// HasPermission / HasPermissionInWorkspace / GetUserRoles /
// GetUserRolesInWorkspace. Only the framework surface (*fiber.Ctx) differs.
type AuthorizationMiddleware struct {
	authorizationService ports.Authorizer
}

// NewAuthorizationMiddleware creates a new authorization middleware instance.
func NewAuthorizationMiddleware(authorizationService ports.Authorizer) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authorizationService: authorizationService,
	}
}

// RequirePermission creates middleware that requires a specific permission.
func (m *AuthorizationMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable.
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware).
		userID, _, ok := GetUserFromContext(c.UserContext())
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
		}

		// Check permission.
		authorized, err := m.authorizationService.HasPermission(c.UserContext(), userID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Authorization check failed"})
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
		}

		return c.Next()
	}
}

// RequireWorkspacePermission creates middleware that requires a workspace-specific permission.
func (m *AuthorizationMiddleware) RequireWorkspacePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable.
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware).
		userID, _, ok := GetUserFromContext(c.UserContext())
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
		}

		// Extract workspace ID from context or URL path.
		workspaceID := GetWorkspaceFromContext(c.UserContext())
		if workspaceID == "" {
			workspaceID = m.extractWorkspaceFromPath(c.Path())
		}

		if workspaceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workspace context required"})
		}

		// Check workspace-specific permission.
		authorized, err := m.authorizationService.HasPermissionInWorkspace(c.UserContext(), userID, workspaceID, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Authorization check failed"})
		}

		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient workspace permissions"})
		}

		return c.Next()
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles.
func (m *AuthorizationMiddleware) RequireAnyRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable.
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware).
		userID, _, ok := GetUserFromContext(c.UserContext())
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
		}

		// Get user roles.
		userRoles, err := m.authorizationService.GetUserRoles(c.UserContext(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get user roles"})
		}

		// Check if user has any of the required roles.
		authorized := m.hasAnyRole(userRoles, roles)
		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient role permissions"})
		}

		return c.Next()
	}
}

// RequireWorkspaceRole creates middleware that requires a specific role within a workspace.
func (m *AuthorizationMiddleware) RequireWorkspaceRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip authorization if service is disabled or unavailable.
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			return c.Next()
		}

		// Extract user ID from context (set by authentication middleware).
		userID, _, ok := GetUserFromContext(c.UserContext())
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
		}

		// Extract workspace ID from context or URL path.
		workspaceID := GetWorkspaceFromContext(c.UserContext())
		if workspaceID == "" {
			workspaceID = m.extractWorkspaceFromPath(c.Path())
		}

		if workspaceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workspace context required"})
		}

		// Get user roles in workspace.
		workspaceRoles, err := m.authorizationService.GetUserRolesInWorkspace(c.UserContext(), userID, workspaceID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get workspace roles"})
		}

		// Check if user has required role in workspace.
		authorized := m.hasRole(workspaceRoles, role)
		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient workspace role"})
		}

		return c.Next()
	}
}

// Helper functions

// extractWorkspaceFromPath attempts to extract workspace ID from URL path.
// Pattern: /api/workspaces/{workspace_id}/...
// Mirrors the vanilla implementation exactly.
func (m *AuthorizationMiddleware) extractWorkspaceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Look for workspace ID in URL patterns.
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// hasAnyRole checks if user has any of the required roles.
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

// hasRole checks if user has a specific role.
func (m *AuthorizationMiddleware) hasRole(userRoles []string, requiredRole string) bool {
	for _, userRole := range userRoles {
		if userRole == requiredRole {
			return true
		}
	}
	return false
}
