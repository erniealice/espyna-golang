//go:build !mock_auth

package mock

import (
	"context"

	"leapfor.xyz/espyna/internal/application/ports"
)

// AllowAllAuthService is a simple authorization service that allows all permissions
// Used as fallback when no auth provider is configured
type AllowAllAuthService struct{}

// NewAllowAllAuth creates a simple allow-all authorization service
func NewAllowAllAuth() ports.AuthorizationService {
	return &AllowAllAuthService{}
}

func (a *AllowAllAuthService) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}

func (a *AllowAllAuthService) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}

func (a *AllowAllAuthService) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	return true, nil
}

func (a *AllowAllAuthService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return []string{"admin"}, nil
}

func (a *AllowAllAuthService) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	return []string{"admin"}, nil
}

func (a *AllowAllAuthService) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	return []string{"default"}, nil
}

func (a *AllowAllAuthService) IsEnabled() bool {
	return true
}
