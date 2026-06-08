package infrastructure

import "context"

// AuthorizationProvider defines the interface for different authorization sources.
// This is a provider lifecycle contract (Name, Initialize, Close) that lives in the
// infrastructure layer, distinct from the Gate 1 RBAC Authorizer interface.
type AuthorizationProvider interface {
	// Name returns the provider name (e.g., "jwt_claims", "database_rbac", "hybrid")
	Name() string

	// HasPermission checks permission using this provider's source
	HasPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasGlobalPermission checks global permission using this provider's source
	HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasPermissionInWorkspace checks workspace-specific permission
	HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error)

	// GetUserRoles retrieves user roles from this provider's source
	GetUserRoles(ctx context.Context, userID string) ([]string, error)

	// GetUserRolesInWorkspace retrieves workspace-specific user roles
	GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error)

	// GetUserWorkspaces retrieves accessible workspaces
	GetUserWorkspaces(ctx context.Context, userID string) ([]string, error)

	// GetUserPermissionCodes returns all effective permission codes for a user
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)

	// IsEnabled returns whether this provider is enabled
	IsEnabled() bool

	// Initialize performs any required setup
	Initialize() error

	// Close performs cleanup
	Close() error
}
