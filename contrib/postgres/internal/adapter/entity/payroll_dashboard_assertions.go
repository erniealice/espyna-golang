//go:build postgresql

package entity

import (
	payrolladapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/payroll"
	payrolldash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/payroll"
)

// Compile-time assertions: every postgres payroll-dashboard repo MUST satisfy
// the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The payroll dashboard composition root performs runtime type assertions
// (see `internal/composition/core/initializers/service.go`) to thread these
// adapters into the service-layer use case. A type assertion silently
// fails — returning `(nil, false)` — when the adapter's method signatures
// drift from the interface, e.g. a return type named
// `payroll.TimeBucket` (the postgres-adapter package-local type) instead of
// `payrolldash.TimeBucket` (the service-layer query-interface type).
//
// **That exact bug shipped Wave B P1.C.1 with `dashboardDeps.AdminRole`
// permanently nil** (codex review P0, 2026-05-20). The role count + top-N
// widget rendered as zero in production. The compile-time assertions below
// are the guard rail that prevents the same trap from re-opening when a
// new adapter method or a renamed type lands in the payroll postgres
// adapter package.
//
// **Each line below catches interface drift while present:** if the adapter's
// method signatures stop satisfying the dashboard interface, the build
// fails with a "does not implement" error pointing at the offending
// repository. Removing a line silently disables its check; treat the
// var-block as an append-only contract (a future static-check pass can
// grep for the expected `var _ <iface>` lines if stronger enforcement is
// needed).
//
// The assertion file lives in the `entity` adapter package (matching the
// admin pilot at `admin_dashboard_assertions.go`) and cross-references the
// payroll adapter package types. Same trap exists for every proto-anchored
// dashboard candidate; each Wave B candidate MUST add equivalent assertions
// when its adapter lands.
var (
	_ payrolldash.PayrollRunDashboardRepository        = (*payrolladapter.PostgresPayrollRunRepository)(nil)
	_ payrolldash.PayrollRemittanceDashboardRepository = (*payrolladapter.PostgresPayrollRemittanceRepository)(nil)
)
