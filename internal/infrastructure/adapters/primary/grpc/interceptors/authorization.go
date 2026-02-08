//go:build grpc_vanilla

package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// AuthorizationInterceptor provides authorization interceptor for gRPC requests
type AuthorizationInterceptor struct {
	authorizationService ports.AuthorizationService
}

// NewAuthorizationInterceptor creates a new authorization interceptor instance
func NewAuthorizationInterceptor(authorizationService ports.AuthorizationService) *AuthorizationInterceptor {
	return &AuthorizationInterceptor{
		authorizationService: authorizationService,
	}
}

// UnaryInterceptor returns a unary server interceptor for authorization
func (i *AuthorizationInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authorization if service is disabled or unavailable
		if i.authorizationService == nil || !i.authorizationService.IsEnabled() {
			return handler(ctx, req)
		}

		// Extract user ID from context (set by authentication interceptor)
		userID, _, ok := GetUserFromContext(ctx)
		if !ok || userID == "" {
			return nil, status.Error(codes.Unauthenticated, "User not authenticated")
		}

		// Check permission based on method
		permission := i.methodToPermission(info.FullMethod)
		if permission == "" {
			// No specific permission required for this method
			return handler(ctx, req)
		}

		// Check permission
		authorized, err := i.authorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			return nil, status.Error(codes.Internal, "Authorization check failed")
		}

		if !authorized {
			return nil, status.Error(codes.PermissionDenied, "Insufficient permissions")
		}

		return handler(ctx, req)
	}
}

// methodToPermission converts gRPC full method to permission string
func (i *AuthorizationInterceptor) methodToPermission(fullMethod string) string {
	// Pattern: /espyna.{domain}.v1.{Resource}Service/{Operation}
	// Permission: {domain}.{resource}.{operation}
	//
	// Example: /espyna.entity.v1.ClientService/Create -> entity.client.create

	parts := strings.Split(strings.Trim(fullMethod, "/"), "/")
	if len(parts) < 2 {
		return ""
	}

	// Parse: espyna.entity.v1.ClientService/Create
	serviceParts := strings.Split(parts[0], ".")
	if len(serviceParts) < 4 {
		return ""
	}

	domain := serviceParts[1]     // entity
	resourceName := serviceParts[3] // ClientService -> client
	operation := parts[1]          // Create

	// Remove "Service" suffix from resource name
	resource := strings.ToLower(strings.TrimSuffix(resourceName, "Service"))

	return strings.ToLower(domain + "." + resource + "." + operation)
}

// RequireWorkspacePermission checks for workspace-specific permissions
func (i *AuthorizationInterceptor) RequireWorkspacePermission(ctx context.Context, permission, workspaceID string) error {
	// Skip authorization if service is disabled or unavailable
	if i.authorizationService == nil || !i.authorizationService.IsEnabled() {
		return nil
	}

	// Extract user ID from context (set by authentication interceptor)
	userID, _, ok := GetUserFromContext(ctx)
	if !ok || userID == "" {
		return status.Error(codes.Unauthenticated, "User not authenticated")
	}

	if workspaceID == "" {
		workspaceID = GetWorkspaceFromContext(ctx)
	}

	if workspaceID == "" {
		return status.Error(codes.InvalidArgument, "Workspace context required")
	}

	// Check workspace-specific permission
	authorized, err := i.authorizationService.HasPermissionInWorkspace(ctx, userID, workspaceID, permission)
	if err != nil {
		return status.Error(codes.Internal, "Authorization check failed")
	}

	if !authorized {
		return status.Error(codes.PermissionDenied, "Insufficient workspace permissions")
	}

	return nil
}

// RequireAnyRole checks if user has any of the specified roles
func (i *AuthorizationInterceptor) RequireAnyRole(ctx context.Context, roles ...string) error {
	// Skip authorization if service is disabled or unavailable
	if i.authorizationService == nil || !i.authorizationService.IsEnabled() {
		return nil
	}

	// Extract user ID from context (set by authentication interceptor)
	userID, _, ok := GetUserFromContext(ctx)
	if !ok || userID == "" {
		return status.Error(codes.Unauthenticated, "User not authenticated")
	}

	// Get user roles
	userRoles, err := i.authorizationService.GetUserRoles(ctx, userID)
	if err != nil {
		return status.Error(codes.Internal, "Failed to get user roles")
	}

	// Check if user has any of the required roles
	authorized := i.hasAnyRole(userRoles, roles)
	if !authorized {
		return status.Error(codes.PermissionDenied, "Insufficient role permissions")
	}

	return nil
}

// RequireWorkspaceRole checks for a specific role within a workspace
func (i *AuthorizationInterceptor) RequireWorkspaceRole(ctx context.Context, role, workspaceID string) error {
	// Skip authorization if service is disabled or unavailable
	if i.authorizationService == nil || !i.authorizationService.IsEnabled() {
		return nil
	}

	// Extract user ID from context (set by authentication interceptor)
	userID, _, ok := GetUserFromContext(ctx)
	if !ok || userID == "" {
		return status.Error(codes.Unauthenticated, "User not authenticated")
	}

	if workspaceID == "" {
		workspaceID = GetWorkspaceFromContext(ctx)
	}

	if workspaceID == "" {
		return status.Error(codes.InvalidArgument, "Workspace context required")
	}

	// Get user roles in workspace
	workspaceRoles, err := i.authorizationService.GetUserRolesInWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return status.Error(codes.Internal, "Failed to get workspace roles")
	}

	// Check if user has required role in workspace
	authorized := i.hasRole(workspaceRoles, role)
	if !authorized {
		return status.Error(codes.PermissionDenied, "Insufficient workspace role")
	}

	return nil
}

// Helper functions

// GetWorkspaceFromContext extracts workspace ID from the context
func GetWorkspaceFromContext(ctx context.Context) string {
	if workspaceID, ok := ctx.Value("workspace_id").(string); ok {
		return workspaceID
	}
	return ""
}

// hasAnyRole checks if user has any of the required roles
func (i *AuthorizationInterceptor) hasAnyRole(userRoles []string, requiredRoles []string) bool {
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
func (i *AuthorizationInterceptor) hasRole(userRoles []string, requiredRole string) bool {
	for _, userRole := range userRoles {
		if userRole == requiredRole {
			return true
		}
	}
	return false
}
