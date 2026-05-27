//go:build sqlserver

package ledger

import (
	ledgerdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/ledger"
)

// Compile-time assertions: every sqlserver ledger-dashboard repo MUST satisfy
// the corresponding service-layer dashboard repository interface.
// Mirrors the postgres gold standard — see ledger_dashboard_assertions.go for the
// full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20).
var (
	_ ledgerdash.AccountDashboardRepository      = (*SQLServerAccountRepository)(nil)
	_ ledgerdash.JournalEntryDashboardRepository = (*SQLServerJournalEntryRepository)(nil)
)
