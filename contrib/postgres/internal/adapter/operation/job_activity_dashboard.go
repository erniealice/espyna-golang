//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: alias TimeBucket
// from the service-layer job dashboard package so the adapter's named return
// type matches the interface.
type TimeBucket = jobdash.TimeBucket

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

// RecentActivity returns the most recent JobActivity rows (ordered by
// entry_date DESC, then date_created DESC) for the workspace's "Recent
// Activity" dashboard widget. Returns sparse protos populated with just the
// columns the widget consumes: id, job_id, quantity, description,
// entry_date (+ entry_date_string).
//
// codex-review-phase1-round2b P1 fix (2026-05-21): adds the missing optional
// repository method so the runtime type assertion in
// `internal/composition/core/initializers/service.go` succeeds, populating
// `dashboardDeps.JobActivityRecent`. Without this method the Recent Activity
// widget rendered permanently empty.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_job_activity_workspace_entrydate
//	  ON job_activity(workspace_id, entry_date DESC)
//	  WHERE active = true;
func (r *PostgresJobActivityRepository) RecentActivity(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*jobactivitypb.JobActivity, error) {
	if r == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	db := jaGetDB(r.dbOps)
	if db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			ja.id,
			ja.job_id,
			ja.quantity,
			ja.description,
			ja.entry_date,
			ja.date_created
		FROM job_activity ja
		WHERE ja.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR ja.workspace_id = $1)
		ORDER BY ja.entry_date DESC NULLS LAST, ja.date_created DESC NULLS LAST
		LIMIT $2`

	rows, err := db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]*jobactivitypb.JobActivity, 0, limit)
	for rows.Next() {
		var (
			id          string
			jobID       string
			quantity    float64
			description sql.NullString
			entryDate   sql.NullTime
			dateCreated sql.NullTime
		)
		if scanErr := rows.Scan(&id, &jobID, &quantity, &description, &entryDate, &dateCreated); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent-activity row: %w", scanErr)
		}
		ja := &jobactivitypb.JobActivity{
			Id:       id,
			JobId:    jobID,
			Quantity: quantity,
			Active:   true,
		}
		if description.Valid {
			d := description.String
			ja.Description = &d
		}
		if entryDate.Valid {
			ts := entryDate.Time.Unix()
			ja.EntryDate = &ts
			s := entryDate.Time.Format("2006-01-02")
			ja.EntryDateString = &s
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.Unix()
			ja.DateCreated = &ts
		}
		out = append(out, ja)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent-activity rows: %w", err)
	}
	return out, nil
}
