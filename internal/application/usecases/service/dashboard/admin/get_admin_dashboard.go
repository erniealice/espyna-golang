package admin

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	admindashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/admin"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// PermissionDashboardRepository is satisfied by PostgresPermissionRepository.
//
// Extension interface — the aggregate `Count` method lives on the postgres
// permission adapter; this package surfaces it as a Go interface the
// composition root assembles via type assertion.
type PermissionDashboardRepository interface {
	Count(ctx context.Context) (int64, error)
}

// RolePermissionCount is one row of the "roles by permission count" table
// widget. Kept as a Go-only repository return type — the service-layer use
// case projects it onto the proto `RolePermissionCount` message.
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

// GetAdminDashboardRepositories groups the per-repository dependencies the
// service-layer admin dashboard composes. Any sub-repository may be nil when
// the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories and
// returns a zero-valued response section for the missing concern.
type GetAdminDashboardRepositories struct {
	Permission        PermissionDashboardRepository
	Role              RoleDashboardRepository
	WorkspaceUser     WorkspaceUserDashboardRepository
	WorkspaceUserRole WorkspaceUserRoleDashboardRepository
}

// GetAdminDashboardServices groups application services. TranslationService
// formats error messages. No AuthorizationService — the dashboard is rendered
// for the active workspace context and the upstream HTTP route is gated by
// session middleware rather than per-entity authcheck.
type GetAdminDashboardServices struct {
	TranslationService ports.TranslationService
}

// GetAdminDashboardUseCase composes the four entity aggregates (permission /
// role / workspace_user / workspace_user_role) into the service-layer admin
// dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the admin-dashboard repository
// composition that previously lived at `usecases/entity/admin/dashboard/`.
// The relocation moves the proto contract out of the entity-driven category
// and into the service-driven category, where it sits alongside the other
// dashboard candidates (Location, Ledger, Equity, Treasury, Payroll, etc.).
type GetAdminDashboardUseCase struct {
	repositories GetAdminDashboardRepositories
	services     GetAdminDashboardServices
}

// NewGetAdminDashboardUseCase wires the use case from grouped dependencies.
func NewGetAdminDashboardUseCase(
	repositories GetAdminDashboardRepositories,
	services GetAdminDashboardServices,
) *GetAdminDashboardUseCase {
	return &GetAdminDashboardUseCase{repositories: repositories, services: services}
}

// Execute fans out the four aggregate queries and assembles the proto
// response. Each branch is nil-safe so the dashboard degrades gracefully on
// non-postgres builds.
func (uc *GetAdminDashboardUseCase) Execute(
	ctx context.Context,
	req *admindashpb.GetAdminDashboardRequest,
) (*admindashpb.GetAdminDashboardResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"admin.dashboard.validation.request_required",
			"admin dashboard: request is required"))
	}

	workspaceID := req.GetWorkspaceId()
	// `now` is used for time-relative stats (currently delegated to
	// CountSinceDays which takes a days int32). Kept for future expansion
	// and parity with the legacy entity-layer use case.
	now := time.Now()
	if req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}
	_ = now

	resp := &admindashpb.GetAdminDashboardResponse{
		Success: true,
		Stats:   &admindashpb.AdminStats{},
	}

	// 4a. Workspace user count + per-role tally.
	if uc.repositories.WorkspaceUser != nil {
		n, err := uc.repositories.WorkspaceUser.CountByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.WorkspaceUsers = n

		byRole, err := uc.repositories.WorkspaceUser.UsersPerRole(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.UsersPerRole = byRole
	}

	// 4b. Role count + top-by-permission-count.
	if uc.repositories.Role != nil {
		n, err := uc.repositories.Role.Count(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.Roles = n

		top, err := uc.repositories.Role.TopByPermissionCount(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		for _, r := range top {
			resp.TopRolesByPerms = append(resp.TopRolesByPerms, &admindashpb.RolePermissionCount{
				RoleId:          r.RoleID,
				RoleName:        r.RoleName,
				PermissionCount: r.PermissionCount,
			})
		}
	}

	// 4c. Permission count — system-level.
	if uc.repositories.Permission != nil {
		n, err := uc.repositories.Permission.Count(ctx)
		if err != nil {
			return nil, err
		}
		resp.Stats.Permissions = n
	}

	// 4d. Recent assignments + 7d count.
	if uc.repositories.WorkspaceUserRole != nil {
		recent, err := uc.repositories.WorkspaceUserRole.RecentAssignments(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.RecentAssignments = recent

		n, err := uc.repositories.WorkspaceUserRole.CountSinceDays(ctx, workspaceID, 7)
		if err != nil {
			return nil, err
		}
		resp.Stats.RecentRoleChanges7D = n
	}

	return resp, nil
}
