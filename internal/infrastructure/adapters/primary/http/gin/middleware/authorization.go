//go:build gin

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"leapfor.xyz/espyna/internal/application/ports"
)

// AuthorizationMiddleware provides authorization middleware for Gin requests
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
func (m *AuthorizationMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			c.Next()
			return
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromGinContext(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Check permission
		authorized, err := m.authorizationService.HasPermission(c.Request.Context(), userID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
			c.Abort()
			return
		}

		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWorkspacePermission creates middleware that requires a workspace-specific permission
func (m *AuthorizationMiddleware) RequireWorkspacePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			c.Next()
			return
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromGinContext(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Extract workspace ID from context or URL parameters
		workspaceID := GetWorkspaceFromContext(c)
		if workspaceID == "" {
			workspaceID = c.Param("workspace_id")
		}

		if workspaceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace context required"})
			c.Abort()
			return
		}

		// Check workspace-specific permission
		authorized, err := m.authorizationService.HasPermissionInWorkspace(c.Request.Context(), userID, workspaceID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
			c.Abort()
			return
		}

		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient workspace permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles
func (m *AuthorizationMiddleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			c.Next()
			return
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromGinContext(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Get user roles
		userRoles, err := m.authorizationService.GetUserRoles(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user roles"})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		authorized := m.hasAnyRole(userRoles, roles)
		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient role permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireWorkspaceRole creates middleware that requires a specific role within a workspace
func (m *AuthorizationMiddleware) RequireWorkspaceRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization if service is disabled or unavailable
		if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
			c.Next()
			return
		}

		// Extract user ID from context (set by authentication middleware)
		userID := GetUserIDFromGinContext(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Extract workspace ID from context or URL parameters
		workspaceID := GetWorkspaceFromContext(c)
		if workspaceID == "" {
			workspaceID = c.Param("workspace_id")
		}

		if workspaceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace context required"})
			c.Abort()
			return
		}

		// Get user roles in workspace
		workspaceRoles, err := m.authorizationService.GetUserRolesInWorkspace(c.Request.Context(), userID, workspaceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workspace roles"})
			c.Abort()
			return
		}

		// Check if user has required role in workspace
		authorized := m.hasRole(workspaceRoles, role)
		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient workspace role"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions

// GetUserIDFromGinContext extracts user ID from the Gin context
// This should be set by the authentication middleware
func GetUserIDFromGinContext(c *gin.Context) string {
	// Check for "uid" (set by authentication middleware)
	if uid, exists := c.Get("uid"); exists {
		if uidStr, ok := uid.(string); ok {
			return uidStr
		}
	}

	// Fallback to "user_id" if available
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}

	return ""
}

// GetWorkspaceFromContext extracts workspace ID from the Gin context
func GetWorkspaceFromContext(c *gin.Context) string {
	if workspaceID, exists := c.Get("workspace_id"); exists {
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
