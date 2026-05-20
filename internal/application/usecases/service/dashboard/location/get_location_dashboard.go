package location

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/location"
)

// LocationAreaCount is one row of the "top areas by location count" widget.
// Kept as a Go-only repository return type — the service-layer use case
// projects it onto the proto `LocationAreaCount` message.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `LocationAreaDashboardRepository` adapter MUST return EXACTLY
// this named type (via `type LocationAreaCount = location.LocationAreaCount`
// alias on the adapter side). Returning the adapter package's own
// `entity.LocationAreaCount` would silently fail the runtime type assertion
// in `initializers/service.go` (Go interface satisfaction requires exact
// named return type match). See
// `contrib/postgres/internal/adapter/entity/location_dashboard_assertions.go`
// for the compile-time guard.
type LocationAreaCount struct {
	LocationAreaID   string
	LocationAreaName string
	LocationCount    int64
}

// LocationDashboardRepository is satisfied by PostgresLocationRepository.
//
// Extension interface — the aggregate methods live on the postgres location
// adapter; this package surfaces them as a Go interface the composition root
// assembles via type assertion.
type LocationDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
	CountByRegion(ctx context.Context, workspaceID string) (map[string]int64, error)
	RecentlyAdded(ctx context.Context, workspaceID string, limit int32) ([]*locationpb.Location, error)
}

// LocationAreaDashboardRepository is satisfied by PostgresLocationAreaRepository.
type LocationAreaDashboardRepository interface {
	CountByLocation(ctx context.Context, workspaceID string, limit int32) ([]LocationAreaCount, error)
}

// GetLocationDashboardRepositories groups the per-repository dependencies the
// service-layer location dashboard composes. Any sub-repository may be nil
// when the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories and
// returns a zero-valued response section for the missing concern.
type GetLocationDashboardRepositories struct {
	Location     LocationDashboardRepository
	LocationArea LocationAreaDashboardRepository
}

// GetLocationDashboardServices groups application services. TranslationService
// formats error messages. No AuthorizationService — the dashboard is rendered
// for the active workspace context and the upstream HTTP route is gated by
// session middleware rather than per-entity authcheck (matches Admin pilot
// pattern at `service/dashboard/admin/get_admin_dashboard.go`).
type GetLocationDashboardServices struct {
	TranslationService ports.TranslationService
}

// GetLocationDashboardUseCase composes the location + location_area
// aggregates into the service-layer location dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the location-dashboard repository
// composition that previously lived at `usecases/entity/location/dashboard/`.
// The relocation moves the proto contract out of the entity-driven category
// and into the service-driven category, where it sits alongside the other
// dashboard candidates (Admin, Ledger, Equity, Treasury, Payroll, etc.).
type GetLocationDashboardUseCase struct {
	repositories GetLocationDashboardRepositories
	services     GetLocationDashboardServices
}

// NewGetLocationDashboardUseCase wires the use case from grouped dependencies.
func NewGetLocationDashboardUseCase(
	repositories GetLocationDashboardRepositories,
	services GetLocationDashboardServices,
) *GetLocationDashboardUseCase {
	return &GetLocationDashboardUseCase{repositories: repositories, services: services}
}

// Execute fans out the four aggregate queries and assembles the proto
// response. Each branch is nil-safe so the dashboard degrades gracefully on
// non-postgres builds.
//
// Steps mirror the standard 5-step shape:
//  1. permission — authorization is enforced at the route level (RBAC
//     middleware) for the dashboard URL. No per-aggregate permission check
//     is needed because every aggregate runs in the same workspace and
//     mirrors data already visible via the existing list views.
//  2. input validation
//  3. business rules — workspace_id is forwarded to every adapter call,
//     where the WHERE clause is enforced.
//  4. repo calls
//  5. response assembly.
func (uc *GetLocationDashboardUseCase) Execute(
	ctx context.Context,
	req *locationdashpb.GetLocationDashboardRequest,
) (*locationdashpb.GetLocationDashboardResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"location.dashboard.validation.request_required",
			"location dashboard: request is required"))
	}

	workspaceID := req.GetWorkspaceId()
	// `now` is reserved for future time-range filtering. Currently unused —
	// kept for parity with the legacy entity-layer use case + the proto's
	// optional now_millis field.
	now := time.Now()
	if req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}
	_ = now

	resp := &locationdashpb.GetLocationDashboardResponse{
		Success: true,
		Stats:   &locationdashpb.LocationStats{},
	}

	// 4a. Status counts → drive Total / Active stat cards.
	if uc.repositories.Location != nil {
		statuses, err := uc.repositories.Location.CountByStatus(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.TotalLocations = statuses["total"]
		resp.Stats.ActiveLocations = statuses["active"]
	}

	// 4b. Region (area) breakdown → drives the bar chart and Regions stat.
	if uc.repositories.Location != nil {
		byRegion, err := uc.repositories.Location.CountByRegion(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.LocationsByRegion = byRegion
		resp.Stats.RegionsCount = int64(len(byRegion))
	}

	// 4c. Top areas → drives the table widget and Areas Count stat.
	if uc.repositories.LocationArea != nil {
		top, err := uc.repositories.LocationArea.CountByLocation(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.Stats.AreasCount = int64(len(top))
		for _, a := range top {
			resp.TopAreas = append(resp.TopAreas, &locationdashpb.LocationAreaCount{
				LocationAreaId:   a.LocationAreaID,
				LocationAreaName: a.LocationAreaName,
				LocationCount:    a.LocationCount,
			})
		}
	}

	// 4d. Recent additions → drives the activity-list widget.
	if uc.repositories.Location != nil {
		recent, err := uc.repositories.Location.RecentlyAdded(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.RecentLocations = recent
	}

	return resp, nil
}
