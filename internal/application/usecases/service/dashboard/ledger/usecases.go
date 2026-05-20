// Package ledger hosts the service-driven Ledger dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The ledger candidate (P1.C.3) is a
// proto-anchored absorbing-flat-field candidate — the proto relocation from
// `proto/v1/domain/ledger/dashboard/dashboard.proto` to
// `proto/v1/service/dashboard/ledger/dashboard.proto` mirrors the Admin
// pilot pattern (P1.C.1).
//
// Ledger is a DISTINCT dashboard from Equity (sibling P1.C.4) — both live
// under `proto/v1/service/dashboard/` in their own subdirs. Ledger reads
// across `account + journal_entry`; Equity reads across `equity_account +
// equity_transaction`. Per Q-SDM-DASHBOARD-SHARED-TYPES each owns its own
// `*Stats` message.
//
// The repository composition that previously lived under
// `usecases/ledger/dashboard/` is hosted here directly — the ledger
// dashboard reads across account + journal_entry and has no aggregate
// root of its own, which is the canonical Q7 signal-1 shape
// (cross-entity projection) for service-driven domains. The ledger-layer
// use case at `usecases/ledger/dashboard/` is RETIRED in the same commit
// (per Q-SDM-DASHBOARD-DOWNSTREAM the fycha-golang block reflection
// wiring is rewired to the new typed-field path
// `uc.Service.Dashboard.Ledger.GetLedgerDashboard.Execute`, and the
// flat `ledger.Dashboard` field at `usecases/ledger/usecases.go:77` is
// removed in the same commit).
//
// Wave B P1.C.3 worked example — see docs/wiki/articles/hexagonal-rules.md §8.
package ledger

// UseCases aggregates every service-driven ledger dashboard use case.
type UseCases struct {
	GetLedgerDashboard *GetLedgerDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// ledger package. Flattened layout mirrors the Admin / Equity pilots: one
// composite struct so the umbrella `NewDashboardUseCases` factory in the
// sibling package can pass it through unchanged.
type Deps struct {
	Account      AccountDashboardRepository
	JournalEntry JournalEntryDashboardRepository
}

// NewUseCases wires every ledger-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetLedgerDashboard: NewGetLedgerDashboardUseCase(
			GetLedgerDashboardRepositories{
				Account:      deps.Account,
				JournalEntry: deps.JournalEntry,
			},
		),
	}
}
