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
//
// Phase 0i: Execute takes/returns proto types (GetAdminDashboardRequest /
// GetAdminDashboardResponse). The old Go-struct Request/Response/AdminStats/
// RolePermissionCount are deleted — proto-generated types replace them.
package dashboard

import (
	"context"
	"errors"
	"time"

	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
	dashboardpb         "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin/dashboard"
)

// PermissionDashboardRepository is satisfied by PostgresPermissionRepository.
type PermissionDashboardRepository interface {
	Count(ctx context.Context) (int64, error)
}

// RolePermissionCount is one row of the "roles by permission count"
// table widget. Kept as a Go-only type because it is an output of
// RoleDashboardRepository (the postgres adapter returns this shape).
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

// Execute fans out the aggregate queries and assembles the proto response.
func (uc *GetAdminDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetAdminDashboardRequest,
) (*dashboardpb.GetAdminDashboardResponse, error) {
	if req == nil {
		return nil, errors.New("admin dashboard: request is required")
	}

	workspaceID := req.GetWorkspaceId()
	// now is used for time-relative stats (e.g. 7-day activity counts).
	// The proto carries now_millis; zero means use server time.
	now := time.Now()
	if req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}
	_ = now // currently used only for CountSinceDays which takes a days int32

	resp := &dashboardpb.GetAdminDashboardResponse{
		Success: true,
		Stats:   &dashboardpb.AdminStats{},
	}

	// 4a. Workspace user count.
	if uc.workspaceUser != nil {
		n, err := uc.workspaceUser.CountByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.WorkspaceUsers = n

		byRole, err := uc.workspaceUser.UsersPerRole(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.UsersPerRole = byRole
	}

	// 4b. Role count + top-by-permission-count.
	if uc.role != nil {
		n, err := uc.role.Count(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.Roles = n

		top, err := uc.role.TopByPermissionCount(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		for _, r := range top {
			resp.TopRolesByPerms = append(resp.TopRolesByPerms, &dashboardpb.RolePermissionCount{
				RoleId:          r.RoleID,
				RoleName:        r.RoleName,
				PermissionCount: r.PermissionCount,
			})
		}
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
		recent, err := uc.workspaceUserRole.RecentAssignments(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.RecentAssignments = recent

		n, err := uc.workspaceUserRole.CountSinceDays(ctx, workspaceID, 7)
		if err != nil {
			return nil, err
		}
		resp.Stats.RecentRoleChanges7D = n
	}

	return resp, nil
}
