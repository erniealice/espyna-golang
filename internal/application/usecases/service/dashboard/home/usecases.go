// Package home hosts the service-driven Home dashboard use cases.
//
// Two operations previously lived as raw-SQL closures in the service-admin
// composition layer (dashboard.go) and in entydad's service/dashboard/data.go:
//
//   - GetHomeDashboard: workspace-scoped user stats, recent activity, and
//     chart data for the identity home dashboard.
//   - ListUsersByRole: users assigned to a given role (role detail page).
//
// Both queries span workspace_user + workspace_user_role + user + role and
// have no aggregate root — canonical Q7 signal-3 shape for service-driven.
// This package is the espyna-native replacement, following the same pattern
// as the sibling admin/ package (P1.C.1).
//
// Return types are Go-only structs (not proto). The composition layer maps
// them to entydad view types (roleusers.UserByRole, userdashboard.DashboardData)
// at the boundary.
package home

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// UseCases aggregates every service-driven home dashboard use case.
type UseCases struct {
	GetHomeDashboard *GetHomeDashboardUseCase
	ListUsersByRole  *ListUsersByRoleUseCase
}

// Deps groups the constructor inputs.
type Deps struct {
	DashboardStats    HomeDashboardStatsRepository
	DashboardActivity HomeDashboardActivityRepository
	DashboardChart    HomeDashboardChartRepository
	UsersByRole       UsersByRoleRepository
	Translator        ports.Translator
}

// NewUseCases wires every home-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when deps carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetHomeDashboard: NewGetHomeDashboardUseCase(
			GetHomeDashboardRepositories{
				Stats:    deps.DashboardStats,
				Activity: deps.DashboardActivity,
				Chart:    deps.DashboardChart,
			},
			GetHomeDashboardServices{Translator: deps.Translator},
		),
		ListUsersByRole: NewListUsersByRoleUseCase(
			deps.UsersByRole,
		),
	}
}
