//go:build mock_auth

// mock_auth sibling of authorizer.go. The production PermissionAuthorizer is
// //go:build !mock_auth, so under the mock_auth tag the rbac package would
// otherwise have ZERO Go files — and the untagged import of this package in
// internal/composition/core/usecases.go (`rbacauth "…/secondary/auth/rbac"`)
// would fail to build with "build constraints exclude all Go files". This is
// the exact pattern the sibling `mock` package uses: fallback.go (!mock_auth)
// + authorization.go (mock_auth) keep that package non-empty under both tags.
//
// Under mock_auth the real RBAC backstop is never the selected path:
// getServices selects AllowAll whenever allowAllFallbackPermitted() is true
// (CONFIG_AUTH_PROVIDER == "mock_auth"), and mock builds run with that env.
// This stub exists ONLY to satisfy the symbol so the untagged composition code
// compiles. It delegates to the same allow-all behaviour as a defensive
// default in case it is ever constructed.
package rbac

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
)

// PermissionAuthorizer (mock_auth build) is a symbol-compatible stand-in for
// the production authorizer. It is allow-all so a mock binary that somehow
// reaches it never spuriously denies (mock builds are dev/test only).
type PermissionAuthorizer struct{}

// compile-time assertion: the mock_auth PermissionAuthorizer also satisfies
// ports.Authorizer, keeping the two build variants interchangeable.
var _ ports.Authorizer = (*PermissionAuthorizer)(nil)

// NewPermissionAuthorizer (mock_auth build) ignores the query — mock builds
// never exercise the real RBAC chain — and returns an allow-all stub.
func NewPermissionAuthorizer(_ securityports.PermissionQuery) *PermissionAuthorizer {
	return &PermissionAuthorizer{}
}

func (a *PermissionAuthorizer) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}

func (a *PermissionAuthorizer) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}

func (a *PermissionAuthorizer) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	return true, nil
}

func (a *PermissionAuthorizer) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

func (a *PermissionAuthorizer) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	return []string{}, nil
}

func (a *PermissionAuthorizer) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

func (a *PermissionAuthorizer) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

func (a *PermissionAuthorizer) IsEnabled() bool {
	return true
}
