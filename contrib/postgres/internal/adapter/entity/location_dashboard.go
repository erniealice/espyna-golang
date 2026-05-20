//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	locationdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/location"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// LocationAreaCount is one row of the "top locations by area count" aggregate.
//
// LocationArea represents a geographic grouping (e.g. NCR, North Luzon),
// while individual Locations belong to one area via location.location_area_id.
// Per Phase 4 of the dashboards plan, the location dashboard surfaces
// per-area concentration and per-area location count via this struct.
//
// **Wave B P1.C.2 / Q-SDM-DASHBOARD-COMPILE-ASSERTIONS (LOCKED 2026-05-20):**
// aliased to `locationdash.LocationAreaCount` so the postgres
// `PostgresLocationAreaRepository` adapter's CountByLocation method
// directly satisfies [locationdash.LocationAreaDashboardRepository]. Go's
// interface satisfaction requires EXACT named-type matches; without this
// alias the adapter would return its own named `entity.LocationAreaCount`,
// silently failing the type assertion in `initializers/service.go` (the
// exact bug that shipped with P1.C.1 Admin; see `role_dashboard.go:46` for
// the equivalent fix). The `location_dashboard_assertions.go` sibling file
// enforces the satisfaction at compile time.
type LocationAreaCount = locationdash.LocationAreaCount

// CountByStatus returns a map of status → count for locations in the
// workspace. Locations have a boolean `active` flag (no enum status), so
// the keys returned are "active" and "inactive".
//
// Workspace isolation: WHERE workspace_id = $1 is applied first.
func (r *PostgresLocationRepository) CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN active = true THEN 1 ELSE 0 END), 0) AS active_count,
			COALESCE(SUM(CASE WHEN active = false THEN 1 ELSE 0 END), 0) AS inactive_count,
			COUNT(*) AS total
		FROM %s
		WHERE ($1::text IS NULL OR $1::text = '' OR workspace_id = $1)
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, workspaceID)

	var activeCount, inactiveCount, total int64
	if err := row.Scan(&activeCount, &inactiveCount, &total); err != nil {
		if err == sql.ErrNoRows {
			return map[string]int64{"active": 0, "inactive": 0, "total": 0}, nil
		}
		return nil, fmt.Errorf("failed to count locations by status: %w", err)
	}

	return map[string]int64{
		"active":   activeCount,
		"inactive": inactiveCount,
		"total":    total,
	}, nil
}

// CountByRegion returns a map of "region" (location area name) → count of
// locations in the workspace. Locations without an area are bucketed under
// "Unassigned".
//
// Note: the Location proto has no `region` field. Per the dashboard plan,
// when a region column is unavailable we GROUP BY the closest geographic
// category column — `location_area_id` joined to `location_area.name`.
//
// Workspace isolation: WHERE workspace_id = $1 is applied first on the
// location table; location_area itself is also workspace-scoped.
func (r *PostgresLocationRepository) CountByRegion(ctx context.Context, workspaceID string) (map[string]int64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(la.name, 'Unassigned') AS area_name,
			COUNT(l.id) AS cnt
		FROM %s l
		LEFT JOIN location_area la ON l.location_area_id = la.id
		WHERE ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
		GROUP BY area_name
		ORDER BY cnt DESC
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to count locations by region: %w", err)
	}
	defer rows.Close()

	result := map[string]int64{}
	for rows.Next() {
		var name string
		var cnt int64
		if err := rows.Scan(&name, &cnt); err != nil {
			return nil, fmt.Errorf("failed to scan region row: %w", err)
		}
		result[name] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating region rows: %w", err)
	}
	return result, nil
}

// RecentlyAdded returns the most-recently-created locations in the workspace,
// ordered by date_created DESC. Soft-deleted rows (active=false) are still
// included — the dashboard surfaces *all* recent additions.
//
// Workspace isolation: WHERE workspace_id = $1.
func (r *PostgresLocationRepository) RecentlyAdded(ctx context.Context, workspaceID string, limit int32) ([]*locationpb.Location, error) {
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			name,
			COALESCE(address, '') AS address,
			active,
			COALESCE(timezone, 'Asia/Manila') AS timezone,
			location_area_id,
			date_created,
			date_modified
		FROM %s
		WHERE ($1::text IS NULL OR $1::text = '' OR workspace_id = $1)
		ORDER BY date_created DESC NULLS LAST
		LIMIT $2
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent locations: %w", err)
	}
	defer rows.Close()

	var out []*locationpb.Location
	for rows.Next() {
		var (
			id             string
			name           string
			address        string
			active         bool
			timezone       string
			locationAreaID *string
			dateCreated    *time.Time
			dateModified   *time.Time
		)
		if err := rows.Scan(&id, &name, &address, &active, &timezone, &locationAreaID, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("failed to scan recent location: %w", err)
		}

		loc := &locationpb.Location{
			Id:       id,
			Name:     name,
			Address:  address,
			Active:   active,
			Timezone: &timezone,
		}
		if locationAreaID != nil {
			loc.LocationAreaId = locationAreaID
		}
		if dateCreated != nil && !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			loc.DateCreated = &ts
			s := dateCreated.Format(time.RFC3339)
			loc.DateCreatedString = &s
		}
		if dateModified != nil && !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			loc.DateModified = &ts
			s := dateModified.Format(time.RFC3339)
			loc.DateModifiedString = &s
		}
		out = append(out, loc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent locations: %w", err)
	}
	return out, nil
}
