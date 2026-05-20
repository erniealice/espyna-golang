//go:build postgresql

package treasury

import (
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// Compile-time assertions: every postgres treasury-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The treasury dashboard composition root performs runtime type assertions
// (see `internal/composition/core/initializers/service.go`) to thread these
// adapters into the service-layer use cases. A type assertion silently
// fails — returning `(nil, false)` — when the adapter's method signatures
// drift from the interface, e.g. a return type named `treasury.LoanSlice`
// (the postgres-adapter package-local type) instead of
// `treasurydash.LoanSlice` (the service-layer query-interface type).
//
// **That exact bug shipped Wave B P1.C.1 with `dashboardDeps.AdminRole`
// permanently nil** (codex review P0, 2026-05-20). The role count + top-N
// widget rendered as zero in production. Aliasing the row type fixed the
// signature; THESE compile-time assertions are the guard rail that prevents
// the same trap from re-opening when a new adapter method or a renamed
// type lands.
//
// **Each line below catches interface drift while present:** if the adapter's
// method signatures stop satisfying the dashboard interface, the build
// fails with a "does not implement" error pointing at the offending
// repository. Removing a line silently disables its check; treat the
// var-block as an append-only contract (a future static-check pass can
// grep for the expected `var _ <iface>` lines if stronger enforcement is
// needed).
//
// Wave B P1.C.5 unified Loan + Cash candidate (Q-SDM-DASHBOARD-COUNT LOCKED
// 2026-05-20). The two slice repositories (Loan/LoanPayment for Loan;
// Collection for Cash) are guarded together.
var (
	_ treasurydash.LoanDashboardRepository        = (*PostgresLoanRepository)(nil)
	_ treasurydash.LoanPaymentDashboardRepository = (*PostgresLoanPaymentRepository)(nil)
	_ treasurydash.CollectionDashboardRepository  = (*PostgresCollectionRepository)(nil)
)
