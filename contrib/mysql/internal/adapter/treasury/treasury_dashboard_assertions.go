//go:build mysql

package treasury

import (
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// Compile-time assertions: every MySQL treasury-dashboard repo MUST satisfy the
// corresponding service-layer dashboard repository interface.
//
// These mirror the postgres assertions in
// contrib/postgres/internal/adapter/treasury/treasury_dashboard_assertions.go —
// see that file for the full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS,
// LOCKED 2026-05-20).
var (
	_ treasurydash.LoanDashboardRepository        = (*MySQLLoanRepository)(nil)
	_ treasurydash.LoanPaymentDashboardRepository = (*MySQLLoanPaymentRepository)(nil)
	_ treasurydash.CollectionDashboardRepository  = (*MySQLCollectionRepository)(nil)
)
