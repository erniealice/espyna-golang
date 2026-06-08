package security

import "context"

// Authorizer defines the interface for authorization operations
// This interface is framework-agnostic and resides in the application layer
type Authorizer interface {
	// HasPermission checks if a user has a specific permission
	HasPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasGlobalPermission checks if a user has a global/system-wide permission
	// This is equivalent to HasPermission but with clearer naming for global scope
	HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasPermissionInWorkspace checks if a user has a permission within a specific workspace
	HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error)

	// GetUserRoles returns all roles assigned to a user
	GetUserRoles(ctx context.Context, userID string) ([]string, error)

	// GetUserRolesInWorkspace returns user roles within a specific workspace
	GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error)

	// GetUserWorkspaces returns all workspaces a user has access to
	GetUserWorkspaces(ctx context.Context, userID string) ([]string, error)

	// GetUserPermissionCodes returns all effective permission codes for a user (for UI adaptation).
	// Returns only ALLOW'd codes that are not overridden by a DENY.
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)

	// IsEnabled returns whether authorization is enabled
	IsEnabled() bool
}

// NoOpAuthorizer provides a non-operational fallback that allows all actions.
type noOpAuthorizationService struct{}

func (s *noOpAuthorizationService) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) IsEnabled() bool {
	return false
}

func NewNoOpAuthorizer() Authorizer {
	return &noOpAuthorizationService{}
}


