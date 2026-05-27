//go:build mysql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"

	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: alias to the
// service-layer types so Go's exact-named-return-type matching makes
// `var _ jobdash.JobDashboardRepository = (*MySQLJobRepository)(nil)`
// succeed in job_dashboard_assertions.go.
type JobRisk = jobdash.JobRisk

// CountByStatus returns a map of job.status (raw enum string e.g.
// "JOB_STATUS_ACTIVE") to count, restricted to jobs created since `since`
// (when non-zero). Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - $1::text IS NULL OR ... → ? (MySQL, empty string guard)
//   - $2 for since date → ? (second positional arg)
//   - COUNT(*)::bigint → COUNT(*) (MySQL returns int64 natively)
//   - active = true → active = 1
func (r *MySQLJobRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	var (
		rows *sql.Rows
		err  error
	)

	if since.IsZero() {
		// Dialect: $1::text IS NULL OR ... → (? = '' OR j.workspace_id = ?)
		const q = `
			SELECT j.status, COUNT(*)
			FROM job j
			WHERE j.active = 1
			  AND (? = '' OR j.workspace_id = ?)
			GROUP BY j.status`
		rows, err = r.db.QueryContext(ctx, q, workspaceID, workspaceID)
	} else {
		const q = `
			SELECT j.status, COUNT(*)
			FROM job j
			WHERE j.active = 1
			  AND j.date_created >= ?
			  AND (? = '' OR j.workspace_id = ?)
			GROUP BY j.status`
		rows, err = r.db.QueryContext(ctx, q, since, workspaceID, workspaceID)
	}
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 8)
	for rows.Next() {
		var (
			status sql.NullString
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan job count row: %w", scanErr)
		}
		key := ""
		if status.Valid {
			key = status.String
		}
		out[key] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job count rows: %w", err)
	}
	return out, nil
}

// UpcomingDeadlines returns active jobs whose planned_end (falling back to
// due_date) is between NOW() and NOW() + (days × 1 day), ordered by deadline.
// Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - to_timestamp(due_date / 1000.0) → FROM_UNIXTIME(due_date / 1000)
//   - ($2 || ' days')::interval → INTERVAL ? DAY
//   - $1::text IS NULL OR ... → (? = ” OR j.workspace_id = ?)
//   - active = true → active = 1
func (r *MySQLJobRepository) UpcomingDeadlines(
	ctx context.Context,
	workspaceID string,
	days int,
	limit int32,
) ([]*jobpb.Job, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if days <= 0 {
		days = 14
	}
	if limit <= 0 {
		limit = 5
	}

	// Dialect: to_timestamp(due_date / 1000.0) → FROM_UNIXTIME(due_date / 1000)
	// COALESCE(planned_end, FROM_UNIXTIME(due_date/1000)) replaces the postgres-only
	// to_timestamp() function. INTERVAL ? DAY replaces ($N || ' days')::interval.
	const query = `
		SELECT
			j.id,
			j.name,
			j.status,
			j.planned_end,
			j.due_date
		FROM job j
		WHERE j.active = 1
		  AND (? = '' OR j.workspace_id = ?)
		  AND COALESCE(j.planned_end, FROM_UNIXTIME(j.due_date / 1000)) IS NOT NULL
		  AND COALESCE(j.planned_end, FROM_UNIXTIME(j.due_date / 1000)) >= NOW()
		  AND COALESCE(j.planned_end, FROM_UNIXTIME(j.due_date / 1000)) <= DATE_ADD(NOW(), INTERVAL ? DAY)
		ORDER BY COALESCE(j.planned_end, FROM_UNIXTIME(j.due_date / 1000)) ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, days, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]*jobpb.Job, 0, limit)
	for rows.Next() {
		var (
			id         string
			name       string
			status     sql.NullString
			plannedEnd sql.NullTime
			dueDateMs  sql.NullInt64
		)
		if scanErr := rows.Scan(&id, &name, &status, &plannedEnd, &dueDateMs); scanErr != nil {
			return nil, fmt.Errorf("failed to scan upcoming-deadline row: %w", scanErr)
		}
		j := &jobpb.Job{Id: id, Name: name, Active: true}
		if plannedEnd.Valid {
			ms := plannedEnd.Time.UnixMilli()
			j.PlannedEnd = &ms
			s := plannedEnd.Time.Format(time.RFC3339)
			j.PlannedEndString = &s
		}
		if dueDateMs.Valid {
			ms := dueDateMs.Int64
			j.DueDate = &ms
		}
		_ = status // status string not mapped here — view widget shows name + date only
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating upcoming-deadline rows: %w", err)
	}
	return out, nil
}

// TopByCompletionRisk ranks active, not-yet-completed jobs by a simple risk
// score. Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - COALESCE(pc.done * 100.0 / NULLIF(pc.total, 0), 0)::float8 → COALESCE(pc.done * 100.0 / NULLIF(pc.total, 0), 0)
//   - $1::text IS NULL OR ... → (? = ” OR j.workspace_id = ?)
//   - SUM(CASE WHEN ...) stays (FILTER is postgres-only; this already uses CASE)
//   - active = true → active = 1
func (r *MySQLJobRepository) TopByCompletionRisk(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]JobRisk, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		WITH phase_counts AS (
			SELECT
				p.job_id,
				COUNT(*)                                            AS total,
				SUM(CASE WHEN p.status = 'JOB_PHASE_STATUS_COMPLETED' THEN 1 ELSE 0 END) AS done
			FROM job_phase p
			WHERE p.active = 1
			GROUP BY p.job_id
		)
		SELECT
			j.id,
			j.name,
			j.status,
			COALESCE(pc.done * 100.0 / NULLIF(pc.total, 0), 0) AS completion_pct,
			j.planned_end
		FROM job j
		LEFT JOIN phase_counts pc ON pc.job_id = j.id
		WHERE j.active = 1
		  AND (? = '' OR j.workspace_id = ?)
		  AND j.status NOT IN ('JOB_STATUS_COMPLETED', 'JOB_STATUS_CLOSED')
		  AND j.planned_end IS NOT NULL
		ORDER BY j.planned_end ASC, completion_pct ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]JobRisk, 0, limit)
	for rows.Next() {
		var (
			id            string
			name          string
			statusStr     sql.NullString
			completionPct float64
			plannedEnd    sql.NullTime
		)
		if scanErr := rows.Scan(&id, &name, &statusStr, &completionPct, &plannedEnd); scanErr != nil {
			return nil, fmt.Errorf("failed to scan job-risk row: %w", scanErr)
		}
		row := JobRisk{
			JobID:         id,
			Code:          name, // schema has no separate "code"; use name as display label
			Name:          strings.TrimSpace(name),
			CompletionPct: completionPct,
		}
		if plannedEnd.Valid {
			row.DateEnd = plannedEnd.Time
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job-risk rows: %w", err)
	}
	return out, nil
}
