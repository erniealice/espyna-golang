// Package fulfillment hosts the service-driven Fulfillment dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The fulfillment candidate (P1.C.12) is a
// NEW-PROTO-NEEDED candidate — the pre-migration shape lived as Go-only
// Request/Response types at `usecases/fulfillment/dashboard/get_page_data.go`,
// exposed via the flat field `fulfillment.UseCases.Dashboard` at
// `usecases/fulfillment/usecases.go:35` (removed in the same commit per
// Q-SDM-DASHBOARD-DOWNSTREAM).
//
// The repository composition that previously lived under
// `usecases/fulfillment/dashboard/` is hosted here directly — the fulfillment
// dashboard reads from the fulfillment aggregate (with status-event joins on
// the postgres side). The single repository interface surfaces four
// aggregate methods (CountByStatus, AvgFulfillmentTimeDays, RecentExceptions,
// DailyDeliveredLast30) defined by the postgres adapter.
//
// **Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):** the postgres
// adapter ships `contrib/postgres/internal/adapter/fulfillment/fulfillment_dashboard_assertions.go`
// with the compile-time `var _ <iface> = (*<concrete>)(nil)` line for the
// repository interface defined below. This guards against the silent
// type-assertion failure trap that shipped Wave B P1.C.1 with a permanently
// nil Role dashboard repo (codex review P0, 2026-05-20). Additionally,
// `TimeBucket` is aliased on the postgres adapter side (per the named-type
// contract documented on this package) so the adapter's signature exactly
// matches the interface.
//
// Wave B P1.C.12 — see docs/wiki/articles/hexagonal-rules.md §8.
package fulfillment

// UseCases aggregates every service-driven fulfillment dashboard use case.
type UseCases struct {
	GetFulfillmentDashboard *GetFulfillmentDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// fulfillment package. Flattened layout mirrors the Admin pilot: one
// composite struct so the umbrella `NewDashboardUseCases` factory in the
// sibling package can pass it through unchanged.
//
// Fulfillment may be nil when the postgres build tag is inactive (or the
// type assertion in the initializer fails) — the Execute method tolerates
// nil repositories and returns a zero-valued response section for the
// missing concern.
type Deps struct {
	Fulfillment FulfillmentDashboardRepository
}

// NewUseCases wires every fulfillment-dashboard service use case from
// grouped dependencies. Returns a non-nil aggregate even when `deps` carries
// nil repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetFulfillmentDashboard: NewGetFulfillmentDashboardUseCase(
			GetFulfillmentDashboardRepositories{
				Fulfillment: deps.Fulfillment,
			},
		),
	}
}
