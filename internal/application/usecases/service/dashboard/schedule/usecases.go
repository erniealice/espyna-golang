// Package schedule hosts the service-driven Schedule (event) dashboard use
// case sub-aggregate.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// LOCKED) and docs/plan/20260520-service-domain-migration/wave-b-surface-map.md
// §P1.C.7, the Schedule dashboard is a service-driven projection over the
// `event` entity domain — it owns no aggregate root of its own. The proto
// contract lives at `proto/v1/service/dashboard/schedule/dashboard.proto`
// (authored from scratch — there was no prior proto). This package is the
// Layer-7 proto-shaped wrapper over the entity-layer use case at
// `internal/application/usecases/event/dashboard/get_page_data.go`.
//
// **Visibility:** app-visible per the Q-ORCH-2-REFINEMENT lock — the
// downstream callsite at `packages/cyta-golang/block/wiring.go` resolves
// the schedule dashboard via the typed-field path
// `uc.Service.Dashboard.Schedule.*`. Apps cannot use the dynamic registry
// (`service.Get[T]`) because Go's `internal/` rule blocks naming
// `*<X>.UseCases` as the generic type parameter from app code.
//
// **Flat-field absorption:** the existing `event.Dashboard` field at
// `internal/application/usecases/event/usecases.go:49` (a flat
// `*GetScheduleDashboardPageDataUseCase` field on the entity-layer
// `EventUseCases` aggregator) is REMOVED in the same change that introduces
// this service-layer wrapper. The entity-layer use case at
// `usecases/event/dashboard/` is retained as the algorithmic implementation;
// this service-layer wrapper translates the proto messages to/from that
// use case (matching the Admin pilot pattern at `service/dashboard/admin/`).
//
// **First new-proto-needed Wave B exemplar.** Schedule is the first Wave B
// candidate where no prior proto existed — it validates the new-proto
// authoring pattern that subsequent flat-field-only candidates (Expenditure,
// Job, Integration, Product, Fulfillment) will follow.
package schedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/event/dashboard"
)

// UseCases aggregates every service-driven schedule dashboard use case. It
// has a single member (GetScheduleDashboard) today — the same shape used
// by Audit, Security, Auth, and Admin makes future expansion (e.g. capacity
// reports, drill-down queries) mechanical.
type UseCases struct {
	GetScheduleDashboard *GetScheduleDashboardUseCase
}

// Deps groups the constructor inputs the umbrella initializer threads to
// the schedule package. Flattened layout mirrors the admin pilot: one
// composite struct so the umbrella `NewDashboardUseCases` factory in the
// sibling package can pass it through unchanged.
//
// EntityDashboard is the entity-layer use case that owns the actual query
// algorithm (CountToday / CountThisWeek / UpcomingByStartDate / CountByDay
// / CountByTag), which in turn delegates to a postgres-backed repo via the
// EventDashboardRepository port. May be nil under non-postgres builds
// (mock_db, mock_auth) — the wrapper degrades gracefully (empty Response).
type Deps struct {
	EntityDashboard      *eventdashboard.GetScheduleDashboardPageDataUseCase
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// NewUseCases wires every schedule-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries a nil
// EntityDashboard — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetScheduleDashboard: NewGetScheduleDashboardUseCase(
			GetScheduleDashboardRepositories{EntityDashboard: deps.EntityDashboard},
			GetScheduleDashboardServices{
				AuthorizationService: deps.AuthorizationService,
				TranslationService:   deps.TranslationService,
			},
		),
	}
}
