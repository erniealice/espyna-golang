// Package treasury hosts the service-driven Treasury dashboard use cases.
// The "Treasury" candidate is UNIFIED — it bundles two distinct dashboards
// (Loan + Cash) under the same proto package and the same Go package per
// Q-SDM-DASHBOARD-COUNT (LOCKED 2026-05-20: treasury.LoanDashboard +
// treasury.CashDashboard count as TWO flat fields but fold into ONE
// Treasury-unified sub-commit group, P1.C.5).
//
// Per Q-SDM-DASHBOARD-LAYOUT and Q-SDM-DASHBOARD-SHARED-TYPES (LOCKED
// 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases
// under `usecases/service/dashboard/<X>/`. Treasury (P1.C.5) is partly
// proto-anchored (Loan messages relocated from
// `proto/v1/domain/treasury/dashboard/dashboard.proto`) and partly
// new-proto-needed (Cash messages authored from scratch — the entity-
// layer use case at `usecases/treasury/collection/dashboard/` previously
// carried Go-only Request/Response shapes).
//
// The flat fields previously at:
//
//   - `usecases/treasury/usecases.go:89` (LoanDashboard *GetLoanDashboardPageDataUseCase)
//   - `usecases/treasury/usecases.go:90` (CashDashboard *GetCashDashboardPageDataUseCase)
//
// are REMOVED in the same commit. The entity-layer use case packages at
// `usecases/treasury/dashboard/` and `usecases/treasury/collection/dashboard/`
// are RETIRED — this package hosts the composition directly. Per
// Q-SDM-DASHBOARD-DOWNSTREAM the downstream service-admin adapter callsite
// at `apps/service-admin/internal/composition/adapters.go` is rewired to the
// new typed-field paths
// `uc.Service.Dashboard.Treasury.Loan.GetLoanDashboard.Execute` and
// `uc.Service.Dashboard.Treasury.Cash.GetCashDashboard.Execute`.
//
// **`.Loan` + `.Cash` sub-fields:** the umbrella `UseCases` struct exposes
// the two siblings explicitly so apps navigate
// `uc.Service.Dashboard.Treasury.Loan.GetLoanDashboard.Execute` and
// `uc.Service.Dashboard.Treasury.Cash.GetCashDashboard.Execute`. Each
// sub-field is its own use-case aggregate keyed on its half of the
// candidate.
//
// Wave B P1.C.5 worked example — see docs/wiki/articles/hexagonal-rules.md §8.
package treasury

// UseCases aggregates every service-driven treasury dashboard use case as
// `.Loan` + `.Cash` sub-fields. Per Q-SDM-DASHBOARD-COUNT the two slices
// share a candidate but expose distinct use cases.
type UseCases struct {
	Loan *LoanUseCases
	Cash *CashUseCases
}

// LoanUseCases is the Loan slice of the Treasury candidate.
type LoanUseCases struct {
	GetLoanDashboard *GetLoanDashboardUseCase
}

// CashUseCases is the Cash slice of the Treasury candidate.
type CashUseCases struct {
	GetCashDashboard *GetCashDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// treasury package. The struct carries both sub-aggregates' repos in one
// composite (mirrors the per-candidate pattern of Admin/Location/Equity) so
// the umbrella `NewDashboardUseCases` factory can pass it through unchanged.
type Deps struct {
	// Loan sub-aggregate repos.
	Loan        LoanDashboardRepository
	LoanPayment LoanPaymentDashboardRepository

	// Cash sub-aggregate repos.
	Collection CollectionDashboardRepository
}

// NewUseCases wires every treasury-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute methods handle the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		Loan: &LoanUseCases{
			GetLoanDashboard: NewGetLoanDashboardUseCase(
				GetLoanDashboardRepositories{
					Loan:        deps.Loan,
					LoanPayment: deps.LoanPayment,
				},
			),
		},
		Cash: &CashUseCases{
			GetCashDashboard: NewGetCashDashboardUseCase(
				GetCashDashboardRepositories{
					Collection: deps.Collection,
				},
			),
		},
	}
}
