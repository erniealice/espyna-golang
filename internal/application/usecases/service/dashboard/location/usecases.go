// Package location hosts the service-driven Location dashboard use cases.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), each dashboard candidate owns its proto under
// `proto/v1/service/dashboard/<X>/dashboard.proto` and its Go use cases under
// `usecases/service/dashboard/<X>/`. The location candidate (P1.C.2) is the
// second proto-anchored absorbing-flat-field pilot — the proto relocation
// from `proto/v1/domain/entity/location/dashboard/` to
// `proto/v1/service/dashboard/location/` mirrors the Admin pilot pattern
// (P1.C.1) that downstream candidates (Ledger, Equity, Treasury, Payroll)
// follow.
//
// The repository composition that previously lived under
// `usecases/entity/location/dashboard/` is hosted here directly — the
// location dashboard reads across location + location_area and has no
// aggregate root of its own, which is the canonical Q7 signal-1 shape
// (cross-domain read / cross-entity projection) for service-driven
// domains. The entity-layer use case at `usecases/entity/location/dashboard/`
// is retired in the same commit (Q-SDM-DASHBOARD-DOWNSTREAM rewires the
// only callsite at `apps/service-admin/internal/composition/adapters.go`
// to the new `uc.Service.Dashboard.Location.GetLocationDashboard.Execute`).
//
// Wave B P1.C.2 worked example — mirrors P1.C.1 Admin; see
// docs/wiki/articles/hexagonal-rules.md §8.
package location

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// UseCases aggregates every service-driven location dashboard use case.
type UseCases struct {
	GetLocationDashboard *GetLocationDashboardUseCase
}

// Repositories groups the per-repository dependencies. Any field may be nil
// when the postgres build tag is inactive — Execute degrades gracefully.
type Repositories struct {
	Location     LocationDashboardRepository
	LocationArea LocationAreaDashboardRepository
}

// Services groups application services.
type Services struct {
	Translator ports.Translator
}

// Deps groups the constructor inputs the umbrella initializer threads to the
// location package. Flattened layout mirrors the Admin pilot at
// `service/dashboard/admin/usecases.go`: one composite struct so the umbrella
// `NewDashboardUseCases` factory in the sibling package can pass it through
// unchanged.
type Deps struct {
	Location     LocationDashboardRepository
	LocationArea LocationAreaDashboardRepository
	Translator   ports.Translator
}

// NewUseCases wires every location-dashboard service use case from grouped
// dependencies. Returns a non-nil aggregate even when `deps` carries nil
// repositories — Execute handles the degraded case internally.
func NewUseCases(deps *Deps) *UseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &UseCases{
		GetLocationDashboard: NewGetLocationDashboardUseCase(
			GetLocationDashboardRepositories{
				Location:     deps.Location,
				LocationArea: deps.LocationArea,
			},
			GetLocationDashboardServices{Translator: deps.Translator},
		),
	}
}
