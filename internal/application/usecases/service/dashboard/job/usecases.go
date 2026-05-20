// Package job hosts the service-driven Job dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The job candidate (P1.C.9) is a
// NEW-PROTO-NEEDED candidate — the pre-migration shape lived as Go-only
// Request/Response types at `usecases/operation/dashboard/get_page_data.go`,
// exposed via the flat field `operation.UseCases.Dashboard` at
// `usecases/operation/usecases.go:115` (removed in the same commit per
// Q-SDM-DASHBOARD-DOWNSTREAM).
//
// **Candidate-name vs. source-aggregate divergence:** the source aggregate
// is `operation`, but the umbrella exposes this dashboard under
// `Service.Dashboard.Job` (per wave-b-surface-map §P1.C.9) since the
// dashboard is job-centric. The Go package name here, the proto package
// name, and the umbrella field name all use `job`. The postgres adapter
// package stays at `operation` because it owns the job + job_activity
// aggregates together.
//
// The repository composition that previously lived under
// `usecases/operation/dashboard/` is hosted here directly — the job
// dashboard reads across job + job_activity (two aggregates in the operation
// domain). Three separate repository interfaces guard the read concerns:
//
//   - JobDashboardRepository — CountByStatus / UpcomingDeadlines / TopByCompletionRisk
//   - JobActivityDashboardRepository — SumHoursByWeek (the labor-hours trend)
//   - JobActivityRecentRepository — RecentActivity (the optional recent-activity widget)
//
// **Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):** the postgres
// adapter ships `contrib/postgres/internal/adapter/operation/job_dashboard_assertions.go`
// with compile-time `var _ <iface> = (*<concrete>)(nil)` lines for every
// repository interface defined below. This guards against the silent
// type-assertion failure trap that shipped Wave B P1.C.1 with a permanently
// nil Role dashboard repo (codex review P0, 2026-05-20). Additionally,
// `TimeBucket` and `JobRisk` are aliased on the postgres adapter side (per
// the named-type contract documented on this package) so the adapter's
// signatures exactly match the interfaces.
//
// Wave B P1.C.9 — see docs/wiki/articles/hexagonal-rules.md §8.
package job

// UseCases aggregates every service-driven job dashboard use case.
type UseCases struct {
	GetJobDashboard *GetJobDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// job package. Flattened layout mirrors the Admin pilot: one composite struct
// so the umbrella `NewDashboardUseCases` factory in the sibling package can
// pass it through unchanged.
//
// Any sub-repository may be nil when the postgres build tag is inactive (or
// the type assertion in the initializer fails) — the Execute method tolerates
// nil repositories and returns a zero-valued response section for the missing
// concern. JobActivityRecent is structurally optional even on postgres builds
// (the entity-layer use case shipped it as an optional dependency).
type Deps struct {
	Job               JobDashboardRepository
	JobActivity       JobActivityDashboardRepository
	JobActivityRecent JobActivityRecentRepository
}

// NewUseCases wires every job-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetJobDashboard: NewGetJobDashboardUseCase(
			GetJobDashboardRepositories{
				Job:               deps.Job,
				JobActivity:       deps.JobActivity,
				JobActivityRecent: deps.JobActivityRecent,
			},
		),
	}
}
