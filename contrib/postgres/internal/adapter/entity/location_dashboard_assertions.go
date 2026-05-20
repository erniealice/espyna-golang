//go:build postgresql

package entity

import (
	locationdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/location"
)

// Compile-time assertions: every postgres location-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED):
//
// The location dashboard composition root performs runtime type assertions
// (see `internal/composition/core/initializers/service.go`) to thread these
// adapters into the service-layer use case. A type assertion silently
// fails — returning `(nil, false)` — when the adapter's method signatures
// drift from the interface, e.g. a return type named
// `entity.LocationAreaCount` instead of `locationdash.LocationAreaCount`.
//
// **That exact bug shipped Wave B P1.C.1 Admin with `dashboardDeps.AdminRole`
// permanently nil** (codex review P0, 2026-05-20). The role count + top-N
// widget rendered as zero in production. Aliasing the row type fixed the
// signature; THESE compile-time assertions are the guard rail that
// prevents the same trap from re-opening when a new adapter method or a
// renamed type lands.
//
// **Each line below catches interface drift while present:** if the adapter's
// method signatures stop satisfying the dashboard interface, the build
// fails with a "does not implement" error pointing at the offending
// repository. Removing a line silently disables its check; treat the
// var-block as an append-only contract (a future static-check pass can
// grep for the expected `var _ <iface>` lines if stronger enforcement is
// needed).
//
// **Same trap exists for every proto-anchored dashboard candidate**
// (Admin done; Ledger, Equity, Treasury, Payroll pending). Each Wave B
// candidate MUST add equivalent assertions when its adapter lands.
var (
	_ locationdash.LocationDashboardRepository     = (*PostgresLocationRepository)(nil)
	_ locationdash.LocationAreaDashboardRepository = (*PostgresLocationAreaRepository)(nil)
)
