// Package payroll hosts the service-driven Payroll dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. Payroll (P1.C.6) is a proto-anchored
// absorbing-flat-field candidate — the proto relocation from
// `proto/v1/domain/payroll/dashboard/` to `proto/v1/service/dashboard/payroll/`
// and the absorption of the flat `payroll.Dashboard` field at
// `usecases/payroll/usecases.go:62` mirror the canonical Admin pilot pattern
// (see hexagonal-rules.md §8 Wave B P1.C.1 worked example).
//
// The repository composition that previously lived under
// `usecases/payroll/dashboard/` is hosted here directly — the payroll
// dashboard reads across `payroll_run` + `payroll_remittance` aggregates
// (cross-entity projection, the canonical Q7 signal-3 shape for service-
// driven domains). The entity-layer use case package is retired in the
// same commit; the postgres adapter methods (CountByStatus, SumGrossByMonth,
// LatestRun, RecentRuns, SumTotalGrossInPeriod, CountDueWithin,
// UpcomingDeadlines) remain on the postgres adapters and are surfaced
// here as extension interfaces.
//
// **Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):** the postgres
// adapter ships `contrib/postgres/internal/adapter/entity/payroll_dashboard_assertions.go`
// with compile-time `var _ <iface> = (*<concrete>)(nil)` lines for every
// repository interface defined below. This guards against the silent
// type-assertion failure trap that shipped Wave B P1.C.1 with a permanently
// nil Role dashboard repo (codex review P0, 2026-05-20).
//
// Wave B P1.C.6 worked example — see docs/wiki/articles/hexagonal-rules.md §8.
package payroll

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
)

// UseCases aggregates every service-driven payroll dashboard use case.
type UseCases struct {
	GetPayrollDashboard *GetPayrollDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to
// the payroll package. Flattened layout mirrors the Admin pilot: one
// composite struct so the umbrella `NewDashboardUseCases` factory in the
// sibling package can pass it through unchanged.
//
// PayrollRun and PayrollRemittance may each be nil when the postgres build
// tag is inactive (or the type assertion in the initializer fails) — the
// Execute method tolerates nil repositories and returns a zero-valued
// response section for the missing concern.
type Deps struct {
	PayrollRun        PayrollRunDashboardRepository
	PayrollRemittance PayrollRemittanceDashboardRepository
	Translator        ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires every payroll-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetPayrollDashboard: NewGetPayrollDashboardUseCase(
			GetPayrollDashboardRepositories{
				PayrollRun:        deps.PayrollRun,
				PayrollRemittance: deps.PayrollRemittance,
			},
			GetPayrollDashboardServices{Translator: deps.Translator},
		),
	}
}
