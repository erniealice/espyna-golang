//go:build vanilla

package middleware

import (
	"context"
	"net/http"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
)

// AuthorizationMiddleware provides authorization middleware for vanilla HTTP requests
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
func (m *AuthorizationMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authorization if service is disabled or unavailable
			if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user ID from context (set by authentication middleware)
			userID, _, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// Check permission
			authorized, err := m.authorizationService.HasPermission(r.Context(), userID, permission)
			if err != nil {
				http.Error(w, "Authorization check failed", http.StatusInternalServerError)
				return
			}

			if !authorized {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireWorkspacePermission creates middleware that requires a workspace-specific permission
func (m *AuthorizationMiddleware) RequireWorkspacePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authorization if service is disabled or unavailable
			if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user ID from context (set by authentication middleware)
			userID, _, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// Extract workspace ID from context or URL path
			workspaceID := GetWorkspaceFromContext(r.Context())
			if workspaceID == "" {
				workspaceID = m.extractWorkspaceFromPath(r.URL.Path)
			}

			if workspaceID == "" {
				http.Error(w, "Workspace context required", http.StatusBadRequest)
				return
			}

			// Check workspace-specific permission
			authorized, err := m.authorizationService.HasPermissionInWorkspace(r.Context(), userID, workspaceID, permission)
			if err != nil {
				http.Error(w, "Authorization check failed", http.StatusInternalServerError)
				return
			}

			if !authorized {
				http.Error(w, "Insufficient workspace permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles
func (m *AuthorizationMiddleware) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authorization if service is disabled or unavailable
			if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user ID from context (set by authentication middleware)
			userID, _, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// Get user roles
			userRoles, err := m.authorizationService.GetUserRoles(r.Context(), userID)
			if err != nil {
				http.Error(w, "Failed to get user roles", http.StatusInternalServerError)
				return
			}

			// Check if user has any of the required roles
			authorized := m.hasAnyRole(userRoles, roles)
			if !authorized {
				http.Error(w, "Insufficient role permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireWorkspaceRole creates middleware that requires a specific role within a workspace
func (m *AuthorizationMiddleware) RequireWorkspaceRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authorization if service is disabled or unavailable
			if m.authorizationService == nil || !m.authorizationService.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user ID from context (set by authentication middleware)
			userID, _, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// Extract workspace ID from context or URL path
			workspaceID := GetWorkspaceFromContext(r.Context())
			if workspaceID == "" {
				workspaceID = m.extractWorkspaceFromPath(r.URL.Path)
			}

			if workspaceID == "" {
				http.Error(w, "Workspace context required", http.StatusBadRequest)
				return
			}

			// Get user roles in workspace
			workspaceRoles, err := m.authorizationService.GetUserRolesInWorkspace(r.Context(), userID, workspaceID)
			if err != nil {
				http.Error(w, "Failed to get workspace roles", http.StatusInternalServerError)
				return
			}

			// Check if user has required role in workspace
			authorized := m.hasRole(workspaceRoles, role)
			if !authorized {
				http.Error(w, "Insufficient workspace role", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions

// GetWorkspaceFromContext extracts workspace ID from the request context
func GetWorkspaceFromContext(ctx context.Context) string {
	if workspaceID, ok := ctx.Value("workspace_id").(string); ok {
		return workspaceID
	}
	return ""
}

// extractWorkspaceFromPath attempts to extract workspace ID from URL path
// Pattern: /api/workspaces/{workspace_id}/...
func (m *AuthorizationMiddleware) extractWorkspaceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Look for workspace ID in URL patterns
	for i, part := range parts {
		if part == "workspaces" && i+1 < len(parts) {
			return parts[i+1]
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
