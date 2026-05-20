//go:build postgresql

package operation

import (
	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
)

// Compile-time assertions: every postgres job-dashboard repo MUST satisfy the
// corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The job dashboard composition root performs runtime type assertions (see
// `internal/composition/core/initializers/service.go`) to thread these
// adapters into the service-layer use case. A type assertion silently
// fails — returning `(nil, false)` — when the adapter's method signatures
// drift from the interface, e.g. a return type named `operation.JobRisk`
// (the postgres-adapter package-local type) instead of `jobdash.JobRisk`
// (the service-layer query-interface type).
//
// **That exact bug shipped Wave B P1.C.1 with `dashboardDeps.AdminRole`
// permanently nil** (codex review P0, 2026-05-20). The role count + top-N
// widget rendered as zero in production. Aliasing the row type fixed the
// signature; THESE compile-time assertions are the guard rail that
// prevents the same trap from re-opening when a new adapter method or a
// renamed type lands.
//
// **Each line below catches interface drift while present:** if the adapter's
// method signatures stop satisfying the dashboard interface, the build
// fails with a "does not implement" error pointing at the offending
// repository. Removing a line silently disables its check; treat the
// var-block as an append-only contract (a future static-check pass can
// grep for the expected `var _ <iface>` lines if stronger enforcement is
// needed).
//
// **Required adapter-side aliases for this candidate** (Wave B P1.C.9):
//
//   - `type JobRisk = jobdash.JobRisk` (on the job adapter side — used by
//     `PostgresJobRepository.TopByCompletionRisk`)
//   - `type TimeBucket = jobdash.TimeBucket` (on the job_activity adapter
//     side — used by `PostgresJobActivityRepository.SumHoursByWeek`)
//
// Note: the operation adapter package currently defines its own JobRisk and
// TimeBucket types. The candidate name "Job" diverges from the source
// aggregate name "operation" (per wave-b-surface-map §P1.C.9 — the umbrella
// exposes this as Service.Dashboard.Job for clarity); the postgres adapter
// package stays at `operation` because it owns both job + job_activity
// aggregates together.
//
// codex-review-phase1-round2b P1 fix (2026-05-21): the optional
// `JobActivityRecentRepository` is now implemented directly on the
// PostgresJobActivityRepository (see job_activity_dashboard.go
// `RecentActivity`). The third assertion below pins the canonical-pattern
// shape — one postgres adapter satisfies BOTH JobActivity-side dashboard
// interfaces.
var (
	_ jobdash.JobDashboardRepository         = (*PostgresJobRepository)(nil)
	_ jobdash.JobActivityDashboardRepository = (*PostgresJobActivityRepository)(nil)
	_ jobdash.JobActivityRecentRepository    = (*PostgresJobActivityRepository)(nil)
)
