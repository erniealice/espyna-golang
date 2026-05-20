// Package product hosts the service-driven Product dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The product candidate (P1.C.11) is a
// NEW-PROTO-NEEDED candidate — the pre-migration shape lived as Go-only
// Request/Response types at `usecases/product/dashboard/get_page_data.go`,
// exposed via the flat field `product.UseCases.Dashboard *GetServiceDashboardPageDataUseCase`
// at `usecases/product/usecases.go:78` (removed in the same commit per
// Q-SDM-DASHBOARD-DOWNSTREAM).
//
// **Candidate-name divergence:** the source use case is
// `GetServiceDashboardPageDataUseCase` because it filters product_kind="service",
// but per wave-b-surface-map §P1.C.11 the umbrella exposes this dashboard
// under `Service.Dashboard.Product` for clarity (the dashboard surfaces a
// slice of the product catalog, scoped by Kind). The Go package name here,
// the proto package name, and the umbrella field name all use `product`.
//
// The repository composition that previously lived under
// `usecases/product/dashboard/` is hosted here directly — the product
// dashboard reads from the product aggregate (with a kind discriminator).
// The single repository interface surfaces three aggregate methods
// (CountByStatusAndKind, CountByLine, RecentlyListed) defined by the
// postgres adapter.
//
// **Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):** the postgres
// adapter ships `contrib/postgres/internal/adapter/product/product_dashboard_assertions.go`
// with the compile-time `var _ <iface> = (*<concrete>)(nil)` line for the
// repository interface defined below. This guards against the silent
// type-assertion failure trap that shipped Wave B P1.C.1 with a permanently
// nil Role dashboard repo (codex review P0, 2026-05-20).
//
// Wave B P1.C.11 — see docs/wiki/articles/hexagonal-rules.md §8.
package product

// UseCases aggregates every service-driven product dashboard use case.
type UseCases struct {
	GetProductDashboard *GetProductDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// product package. Flattened layout mirrors the Admin pilot: one composite
// struct so the umbrella `NewDashboardUseCases` factory in the sibling package
// can pass it through unchanged.
//
// Product may be nil when the postgres build tag is inactive (or the type
// assertion in the initializer fails) — the Execute method tolerates nil
// repositories and returns a zero-valued response section for the missing
// concern.
type Deps struct {
	Product ProductDashboardRepository
}

// NewUseCases wires every product-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetProductDashboard: NewGetProductDashboardUseCase(
			GetProductDashboardRepositories{
				Product: deps.Product,
			},
		),
	}
}
