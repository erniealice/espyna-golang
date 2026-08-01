// Package outcome_completion hosts the service-driven outcome-completion
// dashboard use case (docs/plan/20260801-persona-home-dashboard, Phase 2).
//
// Per Q-SDM-DASHBOARD-LAYOUT / Q-SDM-DASHBOARD-SHARED-TYPES (both LOCKED
// 2026-05-20) the candidate owns its proto under
// `proto/v1/service/dashboard/outcome_completion/dashboard.proto` and its Go
// use case here. It is a cross-aggregate projection over the operation domain
// (job_task × template_task_criteria × task_outcome × job_phase) — a
// service-driven READ contract only: no entityid registration, no table, no
// write side (plan.md §4.3; the `outcome_completion` permission subject is a
// service-read noun, not an entity — copya.md §C-2).
//
// Q5 lock (2026-08-01): ONE dedicated aggregate RPC — this use case does NOT
// extend GetJobDashboard and does NOT compose raw list reads. Q7 lock: the
// Gate 1 capability is the dedicated code `outcome_completion:read`, seeded +
// tagged {1,2,7}; raw `job_phase:list` / `job_task:list` deliberately stay
// closed to staff — the aggregate exposes derived counts only.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED
// 2026-05-20):** the postgres adapter MUST return EXACTLY the named
// `CategoryPeriodCount` type (via a `type CategoryPeriodCount =
// outcomecompletiondash.CategoryPeriodCount` alias on the adapter side) so
// the runtime type assertion in the composition root
// (`internal/composition/core/initializers/service/dashboard.go`) succeeds.
// The compile-time guard lives beside the adapter query file
// (`contrib/postgres/internal/adapter/operation/outcome_completion_dashboard.go`).
package outcome_completion

import "github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

// UseCases aggregates the service-driven outcome-completion dashboard use cases.
type UseCases struct {
	GetOutcomeCompletionSummary *GetOutcomeCompletionSummaryUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to this
// package. Mirrors the sibling `dashboard/job` package: one composite struct
// the umbrella `NewDashboardUseCases` factory passes through unchanged.
//
// The repository may be nil when the postgres build tag is inactive (or the
// type assertion in the initializer fails) — Execute tolerates a nil
// repository and returns a zero-valued response.
type Deps struct {
	OutcomeCompletion OutcomeCompletionSummaryRepository

	// ActionGatekeeper enforces the Gate 1 `outcome_completion:read`
	// capability (Q7 lock). Nil-safe: the gatekeeper's Check denies by
	// default when nil.
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the outcome-completion dashboard use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries a nil
// repository — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetOutcomeCompletionSummary: NewGetOutcomeCompletionSummaryUseCase(
			GetOutcomeCompletionSummaryRepositories{
				OutcomeCompletion: deps.OutcomeCompletion,
			},
			deps.ActionGatekeeper,
		),
	}
}
