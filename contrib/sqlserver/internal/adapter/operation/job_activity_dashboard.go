//go:build sqlserver

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

// SumHoursByWeek returns one TimeBucket per ISO week in the trailing
// `weeks`-week window, summing JobActivity.quantity (hours) for entries whose
// entry_type is LABOR. Workspace-scoped.
//
// SQL Server differences from postgres gold standard:
//   - generate_series → recursive CTE using DATEADD to produce the weekly series.
//   - date_trunc('week', ...) → DATEADD(day, 1 - DATEPART(dw, ...), CAST(... AS date)).
//   - SUM(x) FILTER (WHERE ...) → SUM(CASE WHEN ... END) (A8 rule).
//   - $N → @pN placeholders.
//   - ::bigint → CAST(... AS bigint).
func (r *SQLServerJobActivityRepository) SumHoursByWeek(
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

	// SQL Server: generate the week series via a recursive CTE.
	// DATEADD(day, 1 - DATEPART(dw, GETUTCDATE()), CAST(GETUTCDATE() AS date)) = ISO week start (Sunday offset).
	const query = `
		WITH week_series AS (
			SELECT
				DATEADD(day, 1 - DATEPART(dw, GETUTCDATE()), CAST(GETUTCDATE() AS date)) AS bucket
			UNION ALL
			SELECT DATEADD(week, -1, bucket)
			FROM week_series
			WHERE DATEADD(week, -1, bucket) > DATEADD(week, -@p2, DATEADD(day, 1 - DATEPART(dw, GETUTCDATE()), CAST(GETUTCDATE() AS date)))
		)
		SELECT
			ws.bucket,
			CAST(COALESCE(SUM(
				CASE WHEN ja.entry_type = 'ENTRY_TYPE_LABOR' THEN ja.quantity ELSE 0 END
			) * 100, 0) AS bigint) AS centi_hours
		FROM week_series ws
		LEFT JOIN job_activity ja
			ON ja.active = 1
			AND DATEADD(day, 1 - DATEPART(dw, ja.entry_date), CAST(ja.entry_date AS date)) = ws.bucket
			AND (@p1 IS NULL OR @p1 = '' OR ja.workspace_id = @p1)
		GROUP BY ws.bucket
		ORDER BY ws.bucket ASC
		OPTION (MAXRECURSION 52)`

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

// RecentActivity returns the most recent JobActivity rows for the workspace's
// "Recent Activity" dashboard widget.
//
// SQL Server differences from postgres gold standard:
//   - NULLS LAST → removed (SQL Server puts NULLs last in ASC by default, which
//     is fine here since DESC gives NULLs first; use CASE WHEN to push them last).
//   - LIMIT → TOP (applied on outer SELECT).
//   - $N → @pN.
func (r *SQLServerJobActivityRepository) RecentActivity(
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

	query := fmt.Sprintf(`
		SELECT TOP %d
			ja.id,
			ja.job_id,
			ja.quantity,
			ja.description,
			ja.entry_date,
			ja.date_created
		FROM job_activity ja
		WHERE ja.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR ja.workspace_id = @p1)
		ORDER BY
			CASE WHEN ja.entry_date IS NULL THEN 1 ELSE 0 END,
			ja.entry_date DESC,
			CASE WHEN ja.date_created IS NULL THEN 1 ELSE 0 END,
			ja.date_created DESC`, limit)

	rows, err := db.QueryContext(ctx, query, workspaceID)
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
