//go:build postgresql

package entity

import (
	"context"
	"fmt"
)

// CountByLocation returns the top-N location areas by location count.
//
// Despite the method name, this is *area* concentration — i.e. for each
// LocationArea (a geographic grouping like a region/zone), how many
// Locations belong to it. This is the natural "Top locations by area
// count" widget specified in the Phase 4 dashboard plan: the existing
// schema has no per-Location "area count" so we surface area-level
// concentration instead.
//
// Workspace isolation: WHERE workspace_id = $1 is applied to both the
// location_area table and the joined location table.
func (r *PostgresLocationAreaRepository) CountByLocation(ctx context.Context, workspaceID string, limit int32) ([]LocationAreaCount, error) {
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		SELECT
			la.id,
			la.name,
			COUNT(l.id) AS location_count
		FROM %s la
		LEFT JOIN location l
			ON l.location_area_id = la.id
			AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
		WHERE ($1::text IS NULL OR $1::text = '' OR la.workspace_id = $1)
		GROUP BY la.id, la.name
		ORDER BY location_count DESC, la.name ASC
		LIMIT $2
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top areas by location count: %w", err)
	}
	defer rows.Close()

	var out []LocationAreaCount
	for rows.Next() {
		var rec LocationAreaCount
		if err := rows.Scan(&rec.LocationAreaID, &rec.LocationAreaName, &rec.LocationCount); err != nil {
			return nil, fmt.Errorf("failed to scan top-areas row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top-areas rows: %w", err)
	}
	return out, nil
}
