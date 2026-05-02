//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TimeBucket is a generic (period, value) tuple for time-series aggregates,
// shared across multiple dashboard methods in the operation adapter package.
//
// Value semantics depend on the producing method:
//   - SumHoursByWeek: hours × 100 (centi-hours) for fixed-precision integer math.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// jaGetDB extracts the raw *sql.DB from the dbOps wrapper. Mirrors the
// pattern used by NewPostgresJobRepository (job.go) and the other adapters
// that need direct query access.
func jaGetDB(dbOps any) *sql.DB {
	if dbOps == nil {
		return nil
	}
	pgOps, ok := dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil
	}
	return pgOps.GetDB()
}

// SumHoursByWeek returns one TimeBucket per ISO week in the trailing
// `weeks`-week window, summing JobActivity.quantity (hours) for entries whose
// entry_type is LABOR. Workspace-scoped.
//
// quantity is FLOAT8 in the source schema; we round to centi-hours
// (× 100, integer) so the dashboard pipeline can stay int64-only end-to-end.
// Divide ÷100 only at chart-render to display fractional hours.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_job_activity_workspace_entrydate_type
//	  ON job_activity(workspace_id, entry_date, entry_type)
//	  WHERE active = true;
func (r *PostgresJobActivityRepository) SumHoursByWeek(
	ctx context.Context,
	workspaceID string,
	weeks int,
) ([]TimeBucket, error) {
	if r == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	db := jaGetDB(r.dbOps)
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if weeks <= 0 {
		weeks = 8
	}

	const query = `
		WITH weeks AS (
			SELECT generate_series(
				date_trunc('week', NOW()) - ($2::int - 1) * interval '1 week',
				date_trunc('week', NOW()),
				interval '1 week'
			) AS bucket
		)
		SELECT
			w.bucket,
			COALESCE(SUM(
				CASE WHEN ja.entry_type = 'ENTRY_TYPE_LABOR' THEN ja.quantity ELSE 0 END
			) * 100, 0)::bigint AS centi_hours
		FROM weeks w
		LEFT JOIN job_activity ja
			ON ja.active = true
			AND date_trunc('week', ja.entry_date) = w.bucket
			AND ($1::text IS NULL OR $1::text = '' OR ja.workspace_id = $1)
		GROUP BY w.bucket
		ORDER BY w.bucket ASC`

	rows, err := db.QueryContext(ctx, query, workspaceID, weeks)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan hours-by-week row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hours-by-week rows: %w", err)
	}
	return out, nil
}
