// Package expenditure hosts the service-driven Expenditure dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The expenditure candidate (P1.C.8) is a
// NEW-PROTO-NEEDED candidate — the pre-migration shape lived as Go-only
// Request/Response types at `usecases/expenditure/dashboard/get_page_data.go:44,62`,
// exposed via the flat field `expenditure.Dashboard *GetExpenditureDashboardPageDataUseCase`
// at `usecases/expenditure/usecases.go:127` (removed in the same commit per
// Q-SDM-DASHBOARD-DOWNSTREAM).
//
// The repository composition that previously lived under
// `usecases/expenditure/dashboard/` is hosted here directly — the expenditure
// dashboard reads from the expenditure aggregate filtered by a kind
// discriminator ("purchase" | "expense"). The single repository interface
// surfaces all six aggregate methods (CountByStatus, SumOpenByStatus,
// TopBySupplier, RecentByDate, SumByMonth, SumByCategory) defined by the
// postgres adapter.
//
// **Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):** the postgres
// adapter ships `contrib/postgres/internal/adapter/expenditure/expenditure_dashboard_assertions.go`
// with the compile-time `var _ <iface> = (*<concrete>)(nil)` line for the
// repository interface defined below. This guards against the silent
// type-assertion failure trap that shipped Wave B P1.C.1 with a permanently
// nil Role dashboard repo (codex review P0, 2026-05-20). Additionally,
// `TimeBucket` and `TopSupplierRow` are aliased on the postgres adapter side
// (per the named-type contract documented on this package) so the adapter's
// signatures exactly match the interface.
//
// Wave B P1.C.8 — see docs/wiki/articles/hexagonal-rules.md §8.
package expenditure

// UseCases aggregates every service-driven expenditure dashboard use case.
type UseCases struct {
	GetExpenditureDashboard *GetExpenditureDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// expenditure package. Flattened layout mirrors the Admin pilot: one composite
// struct so the umbrella `NewDashboardUseCases` factory in the sibling package
// can pass it through unchanged.
//
// Expenditure may be nil when the postgres build tag is inactive (or the type
// assertion in the initializer fails) — the Execute method tolerates nil
// repositories and returns a zero-valued response section for the missing
// concern.
type Deps struct {
	Expenditure ExpenditureDashboardRepository
}

// NewUseCases wires every expenditure-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetExpenditureDashboard: NewGetExpenditureDashboardUseCase(
			GetExpenditureDashboardRepositories{
				Expenditure: deps.Expenditure,
			},
		),
	}
}
