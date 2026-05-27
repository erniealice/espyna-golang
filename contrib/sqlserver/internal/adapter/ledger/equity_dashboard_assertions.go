//go:build sqlserver

package ledger

import (
	equitydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/equity"
)

// Compile-time assertions: every sqlserver equity-dashboard repo MUST satisfy
// the corresponding service-layer dashboard repository interface.
// Mirrors the postgres gold standard — see ledger_dashboard_assertions.go for the
// full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED).
var (
	_ equitydash.EquityAccountDashboardRepository     = (*SQLServerEquityAccountRepository)(nil)
	_ equitydash.EquityTransactionDashboardRepository = (*SQLServerEquityTransactionRepository)(nil)
)
