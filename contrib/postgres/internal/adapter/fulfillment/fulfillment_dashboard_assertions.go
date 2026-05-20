//go:build postgresql

package fulfillment

import (
	fulfillmentdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
)

// Compile-time assertions: every postgres fulfillment-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The fulfillment dashboard composition root performs runtime type
// assertions (see `internal/composition/core/initializers/service.go`) to
// thread this adapter into the service-layer use case. A type assertion
// silently fails — returning `(nil, false)` — when the adapter's method
// signatures drift from the interface, e.g. a return type named
// `fulfillment.TimeBucket` (the postgres-adapter package-local type)
// instead of `fulfillmentdash.TimeBucket` (the service-layer query-
// interface type).
//
// **That exact bug shipped Wave B P1.C.1 with `dashboardDeps.AdminRole`
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
// **Required adapter-side aliases for this candidate** (Wave B P1.C.12):
//
//   - `type TimeBucket = fulfillmentdash.TimeBucket` — used by
//     `PostgresFulfillmentRepository.DailyDeliveredLast30`.
//
// Without this alias, the adapter's `DailyDeliveredLast30` method returns
// `[]fulfillment.TimeBucket` and the runtime type assertion silently fails.
// The assertion below catches the drift at build time.
var (
	_ fulfillmentdash.FulfillmentDashboardRepository = (*PostgresFulfillmentRepository)(nil)
)
