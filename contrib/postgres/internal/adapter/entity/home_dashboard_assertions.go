//go:build postgresql

package entity

import (
	homedash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/home"
)

// Compile-time assertions: every postgres home-dashboard repo MUST satisfy
// the corresponding service-layer dashboard repository interface.
//
// See admin_dashboard_assertions.go for the full rationale and
// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED).
var (
	_ homedash.HomeDashboardStatsRepository    = (*PostgresWorkspaceUserRepository)(nil)
	_ homedash.HomeDashboardActivityRepository = (*PostgresWorkspaceUserRepository)(nil)
	_ homedash.HomeDashboardChartRepository    = (*PostgresWorkspaceUserRepository)(nil)
	_ homedash.UsersByRoleRepository           = (*PostgresWorkspaceUserRoleRepository)(nil)
)
