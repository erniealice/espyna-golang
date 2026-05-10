// Package dashboard wires aggregate queries on the Location and LocationArea
// repositories into a typed page-data response for the entydad location
// dashboard view (Phase 4 of the Pyeza dashboard plan).
//
// The use case is read-only and follows the Get/List use case family per the
// view-vs-usecase boundary article. Unlike the other location use cases, the
// dashboard repositories are *extension interfaces* — the aggregate methods
// are added directly to the postgres adapter (no proto/esqyma changes) and
// surfaced here as the LocationDashboardRepository / LocationAreaDashboardRepository
// interfaces. The container assembles these by type-asserting the postgres
// repositories into these interfaces.
//
// Phase 0i: Execute takes/returns proto types (GetLocationDashboardRequest /
// GetLocationDashboardResponse). The old Go-struct Request/Response/LocationStats
// are deleted — the proto-generated types replace them.
package dashboard

import (
	"context"
	"errors"

	locationpb   "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	dashboardpb  "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location/dashboard"
)

// LocationAreaCount is one row of the "top areas by location count" widget.
// Kept as a Go-only type because it is an output of LocationAreaDashboardRepository
// (the postgres adapter returns this shape). The proto message LocationAreaCount
// in dashboardpb is assembled from it during response translation.
type LocationAreaCount struct {
	LocationAreaID   string
	LocationAreaName string
	LocationCount    int64
}

// LocationDashboardRepository is the extension interface the postgres
// adapter satisfies via methods on PostgresLocationRepository:
// CountByStatus, CountByRegion, RecentlyAdded.
type LocationDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
	CountByRegion(ctx context.Context, workspaceID string) (map[string]int64, error)
	RecentlyAdded(ctx context.Context, workspaceID string, limit int32) ([]*locationpb.Location, error)
}

// LocationAreaDashboardRepository is the extension interface the postgres
// adapter satisfies via methods on PostgresLocationAreaRepository:
// CountByLocation.
type LocationAreaDashboardRepository interface {
	CountByLocation(ctx context.Context, workspaceID string, limit int32) ([]LocationAreaCount, error)
}

// GetLocationDashboardPageDataUseCase composes the location and location_area
// aggregate methods into a single page-data response.
type GetLocationDashboardPageDataUseCase struct {
	location LocationDashboardRepository
	area     LocationAreaDashboardRepository
}

// NewGetLocationDashboardPageDataUseCase constructs the use case from the
// extension repositories.
func NewGetLocationDashboardPageDataUseCase(
	location LocationDashboardRepository,
	area LocationAreaDashboardRepository,
) *GetLocationDashboardPageDataUseCase {
	return &GetLocationDashboardPageDataUseCase{
		location: location,
		area:     area,
	}
}

// Execute fans out the four aggregate queries and assembles the proto response.
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
func (uc *GetLocationDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetLocationDashboardRequest,
) (*dashboardpb.GetLocationDashboardResponse, error) {
	// 2. Input validation
	if req == nil {
		return nil, errors.New("location dashboard: request is required")
	}

	workspaceID := req.GetWorkspaceId()
	resp := &dashboardpb.GetLocationDashboardResponse{
		Success: true,
		Stats:   &dashboardpb.LocationStats{},
	}

	// 4a. Status counts → drive Total / Active stat cards.
	if uc.location != nil {
		statuses, err := uc.location.CountByStatus(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.Stats.TotalLocations = statuses["total"]
		resp.Stats.ActiveLocations = statuses["active"]
	}

	// 4b. Region (area) breakdown → drives the bar chart and Regions stat.
	if uc.location != nil {
		byRegion, err := uc.location.CountByRegion(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		resp.LocationsByRegion = byRegion
		resp.Stats.RegionsCount = int64(len(byRegion))
	}

	// 4c. Top areas → drives the table widget and Areas Count stat.
	if uc.area != nil {
		top, err := uc.area.CountByLocation(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.Stats.AreasCount = int64(len(top))
		for _, a := range top {
			resp.TopAreas = append(resp.TopAreas, &dashboardpb.LocationAreaCount{
				LocationAreaId:   a.LocationAreaID,
				LocationAreaName: a.LocationAreaName,
				LocationCount:    a.LocationCount,
			})
		}
	}

	// 4d. Recent additions → drives the activity-list widget.
	if uc.location != nil {
		recent, err := uc.location.RecentlyAdded(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		resp.RecentLocations = recent
	}

	return resp, nil
}
