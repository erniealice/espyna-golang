// Package admin hosts the service-driven Admin dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The admin candidate (P1.C.1) is the first
// proto-anchored absorbing-flat-field pilot — the proto relocation from
// `proto/v1/domain/entity/admin/dashboard/` to `proto/v1/service/dashboard/admin/`
// validates the pattern downstream candidates (Location, Ledger, Equity,
// Treasury, Payroll) follow.
//
// The repository composition that previously lived under
// `usecases/entity/admin/dashboard/` is hosted here directly — admin dashboard
// reads across permission + role + workspace_user + workspace_user_role and
// has no aggregate root of its own, which is the canonical Q7 signal-3 shape
// for service-driven domains. The entity-layer use case is retired in the
// same commit (Q-SDM-DASHBOARD-DOWNSTREAM rewires the only callsite at
// `apps/service-admin/internal/composition/adapters.go:266` to the new
// `uc.Service.Dashboard.Admin.GetAdminDashboard.Execute`).
//
// Wave B P1.C.1 worked example — see docs/wiki/articles/hexagonal-rules.md §8.
package admin

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// UseCases aggregates every service-driven admin dashboard use case.
type UseCases struct {
	GetAdminDashboard *GetAdminDashboardUseCase
}

// Repositories groups the per-repository dependencies. Any field may be nil
// when the postgres build tag is inactive — Execute degrades gracefully.
type Repositories struct {
	Permission        PermissionDashboardRepository
	Role              RoleDashboardRepository
	WorkspaceUser     WorkspaceUserDashboardRepository
	WorkspaceUserRole WorkspaceUserRoleDashboardRepository
}

// Services groups application services.
type Services struct {
	Translator ports.Translator
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// admin package. Flattened layout mirrors audit/security on `service/`: one
// composite struct so the umbrella `NewDashboardUseCases` factory in the
// sibling package can pass it through unchanged.
type Deps struct {
	Permission        PermissionDashboardRepository
	Role              RoleDashboardRepository
	WorkspaceUser     WorkspaceUserDashboardRepository
	WorkspaceUserRole WorkspaceUserRoleDashboardRepository
	Translator        ports.Translator
}

// NewUseCases wires every admin-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetAdminDashboard: NewGetAdminDashboardUseCase(
			GetAdminDashboardRepositories{
				Permission:        deps.Permission,
				Role:              deps.Role,
				WorkspaceUser:     deps.WorkspaceUser,
				WorkspaceUserRole: deps.WorkspaceUserRole,
			},
			GetAdminDashboardServices{Translator: deps.Translator},
		),
	}
}
