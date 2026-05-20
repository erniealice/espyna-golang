// Package equity hosts the service-driven Equity dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The equity candidate (P1.C.4) is a
// proto-anchored absorbing-flat-field candidate — the proto relocation from
// `proto/v1/domain/ledger/equity_dashboard/` to `proto/v1/service/dashboard/
// equity/` validates the per-dashboard pattern (sibling of Admin/Ledger).
//
// Equity is a DISTINCT dashboard from Ledger (sibling P1.C.3). Per
// Q-SDM-DASHBOARD-SHARED-TYPES, each per-dashboard package owns its own
// `*Stats` / `*Slice` shapes — equity's TotalContributed/ActiveOwners/
// DistributionsYtd/NetMovementYtd differ from ledger's TotalAssets/...
//
// The repository composition that previously lived under
// `usecases/ledger/equity_dashboard/` is hosted here directly — equity
// dashboard reads across equity_account + equity_transaction and has no
// aggregate root of its own, which is the canonical Q7 signal-3 shape for
// service-driven domains. The ledger-layer use case is retired in the same
// commit (Q-SDM-DASHBOARD-DOWNSTREAM rewires the only callsite at
// `apps/service-admin/internal/composition/adapters.go:425` to the new
// `uc.Service.Dashboard.Equity.GetEquityDashboard.Execute`).
//
// Wave B P1.C.4 worked example — see docs/wiki/articles/hexagonal-rules.md §8.
package equity

// UseCases aggregates every service-driven equity dashboard use case.
type UseCases struct {
	GetEquityDashboard *GetEquityDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// equity package. Flattened layout mirrors admin/audit/security on
// `service/`: one composite struct so the umbrella `NewDashboardUseCases`
// factory in the sibling package can pass it through unchanged.
type Deps struct {
	EquityAccount     EquityAccountDashboardRepository
	EquityTransaction EquityTransactionDashboardRepository
}

// NewUseCases wires every equity-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetEquityDashboard: NewGetEquityDashboardUseCase(
			GetEquityDashboardRepositories{
				EquityAccount:     deps.EquityAccount,
				EquityTransaction: deps.EquityTransaction,
			},
		),
	}
}
