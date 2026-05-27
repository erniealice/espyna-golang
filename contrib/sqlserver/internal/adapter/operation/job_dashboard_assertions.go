//go:build sqlserver

package operation

import (
	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
)

// Compile-time assertions: every SQL Server job-dashboard repo MUST satisfy the
// corresponding service-layer dashboard repository interface.
//
// See contrib/postgres/internal/adapter/operation/job_dashboard_assertions.go
// for the full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20).
var (
	_ jobdash.JobDashboardRepository         = (*SQLServerJobRepository)(nil)
	_ jobdash.JobActivityDashboardRepository = (*SQLServerJobActivityRepository)(nil)
	_ jobdash.JobActivityRecentRepository    = (*SQLServerJobActivityRepository)(nil)
)
