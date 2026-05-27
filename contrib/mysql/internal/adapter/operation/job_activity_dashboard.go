//go:build mysql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
)

// TimeBucket is a named-type alias from the service-layer job dashboard package
// so the adapter's named return type matches the interface.
type TimeBucket = jobdash.TimeBucket

// jaGetDB extracts the raw *sql.DB from the dbOps wrapper.
func jaGetDB(dbOps any) *sql.DB {
	if dbOps == nil {
		return nil
	}
	myOps, ok := dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil
	}
	return myOps.GetDB()
}

// SumHoursByWeek returns one TimeBucket per ISO week in the trailing
// `weeks`-week window, summing JobActivity.quantity (hours) for entries whose
// entry_type is LABOR. Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - generate_series(...) → recursive CTE (MySQL has no generate_series; use a number table trick
//     or a recursive CTE to produce N weeks)
//   - date_trunc('week', ...) → DATE_FORMAT(DATE_SUB(date, INTERVAL WEEKDAY(date) DAY), '%Y-%m-%d')
//     equivalent via YEARWEEK / STR_TO_DATE
//   - $1::text IS NULL OR ... → (? = ” OR ...)
//   - COALESCE(SUM(...))::bigint → COALESCE(SUM(...), 0) (MySQL returns int natively)
//   - active = true → active = 1
//   - FILTER (WHERE) → CASE WHEN (already used in postgres source)
//
// quantity × 100 to centi-hours so dashboard pipeline stays int64-only.
func (r *MySQLJobActivityRepository) SumHoursByWeek(
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

	// Dialect: MySQL 8.0 recursive CTE to generate a series of week-start dates.
	// DATE_SUB(CURDATE(), INTERVAL WEEKDAY(CURDATE()) DAY) = Monday of current week.
	// We then step back (weeks-1) additional weeks for the window start.
	query := fmt.Sprintf(`
		WITH RECURSIVE week_series AS (
			SELECT DATE_SUB(CURDATE(), INTERVAL WEEKDAY(CURDATE()) DAY) AS bucket,
			       0 AS n
			UNION ALL
			SELECT DATE_SUB(bucket, INTERVAL 1 WEEK), n + 1
			FROM week_series
			WHERE n < %d - 1
		)
		SELECT
			w.bucket,
			COALESCE(SUM(
				CASE WHEN ja.entry_type = 'ENTRY_TYPE_LABOR' THEN ja.quantity ELSE 0 END
			) * 100, 0) AS centi_hours
		FROM week_series w
		LEFT JOIN job_activity ja
			ON ja.active = 1
			AND DATE_SUB(ja.entry_date, INTERVAL WEEKDAY(ja.entry_date) DAY) = w.bucket
			AND (? = '' OR ja.workspace_id = ?)
		GROUP BY w.bucket
		ORDER BY w.bucket ASC`, weeks)

	rows, err := db.QueryContext(ctx, query, workspaceID, workspaceID)
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

// RecentActivity returns the most recent JobActivity rows for the workspace's
// "Recent Activity" dashboard widget.
//
// Dialect translation from postgres gold standard:
//   - $1::text IS NULL OR ... → (? = ” OR ...)
//   - ORDER BY ... NULLS LAST → ORDER BY ... IS NULL ASC (MySQL)
//   - LIMIT $2 → LIMIT ?
//   - active = true → active = 1
func (r *MySQLJobActivityRepository) RecentActivity(
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

	// Dialect: NULLS LAST → IS NULL ASC (MySQL sorts NULLs first in ASC by default;
	// to put NULLs last for DESC, use: ORDER BY ja.entry_date IS NULL ASC, ja.entry_date DESC)
	const query = `
		SELECT
			ja.id,
			ja.job_id,
			ja.quantity,
			ja.description,
			ja.entry_date,
			ja.date_created
		FROM job_activity ja
		WHERE ja.active = 1
		  AND (? = '' OR ja.workspace_id = ?)
		ORDER BY ja.entry_date IS NULL ASC, ja.entry_date DESC,
		         ja.date_created IS NULL ASC, ja.date_created DESC
		LIMIT ?`

	rows, err := db.QueryContext(ctx, query, workspaceID, workspaceID, limit)
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
