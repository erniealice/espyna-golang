//go:build postgresql

package product

import (
	productdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/product"
)

// Compile-time assertions: every postgres product-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The product dashboard composition root performs runtime type assertions
// (see `internal/composition/core/initializers/service.go`) to thread this
// adapter into the service-layer use case. A type assertion silently
// fails — returning `(nil, false)` — when the adapter's method signatures
// drift from the interface.
//
// **That exact bug shipped Wave B P1.C.1 with `dashboardDeps.AdminRole`
// permanently nil** (codex review P0, 2026-05-20). The role count + top-N
// widget rendered as zero in production. THESE compile-time assertions are
// the guard rail that prevents the same trap from re-opening when a new
// adapter method or a renamed type lands.
//
// **Each line below catches interface drift while present:** if the adapter's
// method signatures stop satisfying the dashboard interface, the build
// fails with a "does not implement" error pointing at the offending
// repository. Removing a line silently disables its check; treat the
// var-block as an append-only contract (a future static-check pass can
// grep for the expected `var _ <iface>` lines if stronger enforcement is
// needed).
//
// The product dashboard's `ProductDashboardRepository` interface returns
// only proto types (`map[string]int64`, `[]*productpb.Product`) — there are
// no Go-only row structs to alias today. Future expansion (e.g. a top-
// revenue join returning a slice row) will need the same named-type
// contract documented on
// `usecases/service/dashboard/product/get_product_dashboard.go`.
var (
	_ productdash.ProductDashboardRepository = (*PostgresProductRepository)(nil)
)
