// Package dashboard composes aggregate queries across the
// permission/role/workspace_user/workspace_user_role repositories into a
// typed response for the entydad admin app dashboard view (Phase 4b of the
// Pyeza dashboard plan).
//
// The admin app is a *composite* — its dashboard surfaces aggregates across
// 5 entities (permission, role, workspace, workspace_user,
// workspace_user_role). This use case lives under
// usecases/entity/admin/dashboard/ to mirror the existing entity/admin
// directory shape, even though the admin entity itself is unrelated to the
// admin sidebar app.
//
// Each repository dependency is an *extension interface* (see
// LocationDashboardRepository for prior art) — the aggregate methods are
// added directly to the postgres adapters with no proto/esqyma changes,
// and surfaced here as Go interfaces the container assembles via type
// assertion.
package dashboard

import (
	"context"
	"errors"
	"time"

	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// PermissionDashboardRepository is satisfied by PostgresPermissionRepository.
type PermissionDashboardRepository interface {
	Count(ctx context.Context) (int64, error)
}

// RolePermissionCount is one row of the "roles by permission count"
// table widget. Mirrors the postgres adapter's row type.
type RolePermissionCount struct {
	RoleID          string
	RoleName        string
	PermissionCount int64
}

// RoleDashboardRepository is satisfied by PostgresRoleRepository.
type RoleDashboardRepository interface {
	Count(ctx context.Context, workspaceID string) (int64, error)
	TopByPermissionCount(ctx context.Context, workspaceID string, limit int32) ([]RolePermissionCount, error)
}

// WorkspaceUserDashboardRepository is satisfied by PostgresWorkspaceUserRepository.
type WorkspaceUserDashboardRepository interface {
	CountByWorkspace(ctx context.Context, workspaceID string) (int64, error)
	UsersPerRole(ctx context.Context, workspaceID string) (map[string]int64, error)
}

// WorkspaceUserRoleDashboardRepository is satisfied by
// PostgresWorkspaceUserRoleRepository.
type WorkspaceUserRoleDashboardRepository interface {
	RecentAssignments(ctx context.Context, workspaceID string, limit int32) ([]*workspaceuserrolepb.WorkspaceUserRole, error)
	CountSinceDays(ctx context.Context, workspaceID string, days int32) (int64, error)
}

// GetAdminDashboardPageDataRequest is the input for the admin dashboard
// use case.
type GetAdminDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// AdminStats are the four stat cards: Workspace Users / Roles / Permissions
// / Recent Role Changes (7d).
type AdminStats struct {
	WorkspaceUsers     int64
	Roles              int64
	Permissions        int64
	RecentRoleChanges7d int64
}

// GetAdminDashboardPageDataResponse is the projected aggregate set the view
// layer renders into the pyeza DashboardData.
type GetAdminDashboardPageDataResponse struct {
	Stats              AdminStats
	UsersPerRole       map[string]int64
	TopRolesByPerms    []RolePermissionCount
	RecentAssignments  []*workspaceuserrolepb.WorkspaceUserRole
}

// GetAdminDashboardPageDataUseCase composes the four entity aggregates.
type GetAdminDashboardPageDataUseCase struct {
	permission        PermissionDashboardRepository
	role              RoleDashboardRepository
	workspaceUser     WorkspaceUserDashboardRepository
	workspaceUserRole WorkspaceUserRoleDashboardRepository
}

// NewGetAdminDashboardPageDataUseCase constructs the use case.
func NewGetAdminDashboardPageDataUseCase(
	permission PermissionDashboardRepository,
	role RoleDashboardRepository,
	workspaceUser WorkspaceUserDashboardRepository,
	workspaceUserRole WorkspaceUserRoleDashboardRepository,
) *GetAdminDashboardPageDataUseCase {
	return &GetAdminDashboardPageDataUseCase{
		permission:        permission,
		role:              role,
		workspaceUser:     workspaceUser,
		workspaceUserRole: workspaceUserRole,
	}
}

// Execute fans out the aggregate queries and assembles the response.
func (uc *GetAdminDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetAdminDashboardPageDataRequest,
) (*GetAdminDashboardPageDataResponse, error) {
	if req == nil {
		return nil, errors.New("admin dashboard: request is required")
	}

	resp := &GetAdminDashboardPageDataResponse{}

	// 4a. Workspace user count.
	if uc.workspaceUser != nil {
		n, err := uc.workspaceUser.CountByWorkspace(ctx, req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.WorkspaceUsers = n

		byRole, err := uc.workspaceUser.UsersPerRole(ctx, req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		resp.UsersPerRole = byRole
	}

	// 4b. Role count + top-by-permission-count.
	if uc.role != nil {
		n, err := uc.role.Count(ctx, req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.Roles = n

		top, err := uc.role.TopByPermissionCount(ctx, req.WorkspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.TopRolesByPerms = top
	}

	// 4c. Permission count — system-level.
	if uc.permission != nil {
		n, err := uc.permission.Count(ctx)
		if err != nil {
			return nil, err
		}
		resp.Stats.Permissions = n
	}

	// 4d. Recent assignments + 7d count.
	if uc.workspaceUserRole != nil {
		recent, err := uc.workspaceUserRole.RecentAssignments(ctx, req.WorkspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.RecentAssignments = recent

		n, err := uc.workspaceUserRole.CountSinceDays(ctx, req.WorkspaceID, 7)
		if err != nil {
			return nil, err
		}
		resp.Stats.RecentRoleChanges7d = n
	}

	return resp, nil
}
