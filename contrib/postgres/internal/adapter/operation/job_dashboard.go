//go:build postgresql

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
// `var _ jobdash.JobDashboardRepository = (*PostgresJobRepository)(nil)`
// succeed in job_dashboard_assertions.go.
type JobRisk = jobdash.JobRisk

// CountByStatus returns a map of job.status (raw enum string e.g.
// "JOB_STATUS_ACTIVE") to count, restricted to jobs created since `since`
// (when non-zero). Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_job_workspace_status_created
//	  ON job(workspace_id, status, date_created);
func (r *PostgresJobRepository) CountByStatus(
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
		const q = `
			SELECT j.status, COUNT(*)::bigint
			FROM job j
			WHERE j.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR j.workspace_id = $1)
			GROUP BY j.status`
		rows, err = r.db.QueryContext(ctx, q, workspaceID)
	} else {
		const q = `
			SELECT j.status, COUNT(*)::bigint
			FROM job j
			WHERE j.active = true
			  AND j.date_created >= $2
			  AND ($1::text IS NULL OR $1::text = '' OR j.workspace_id = $1)
			GROUP BY j.status`
		rows, err = r.db.QueryContext(ctx, q, workspaceID, since)
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
// Workspace-scoped. Returns sparse Job protos populated with the columns
// needed by the dashboard widget (id, name, status, planned_end, due_date).
func (r *PostgresJobRepository) UpcomingDeadlines(
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

	// codex-review-phase1-round2b P2 fix (2026-05-21): implement the comment's
	// claim that planned_end falls back to due_date. due_date is stored as
	// epoch millis (bigint), so to_timestamp(due_date / 1000) converts it to
	// a timestamp for comparison; COALESCE picks planned_end first when set.
	// This means due-date-only jobs (with NULL planned_end) now surface in
	// the upcoming-deadlines widget.
	const query = `
		SELECT
			j.id,
			j.name,
			j.status,
			j.planned_end,
			j.due_date
		FROM job j
		WHERE j.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR j.workspace_id = $1)
		  AND COALESCE(j.planned_end, to_timestamp(j.due_date / 1000.0)) IS NOT NULL
		  AND COALESCE(j.planned_end, to_timestamp(j.due_date / 1000.0)) >= NOW()
		  AND COALESCE(j.planned_end, to_timestamp(j.due_date / 1000.0)) <= NOW() + ($2 || ' days')::interval
		ORDER BY COALESCE(j.planned_end, to_timestamp(j.due_date / 1000.0)) ASC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, days, limit)
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
// score: the smaller (planned_end − NOW()) the higher the score; ties broken
// by lower completion_pct ASC (we approximate completion_pct as the share of
// completed phases since the schema does not store a numeric completion).
//
// Workspace-scoped. Returns at most `limit` rows.
func (r *PostgresJobRepository) TopByCompletionRisk(
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

	// Approximation: completion_pct = (completed_phases / total_phases) × 100.
	// Risk ordered by planned_end ASC (closest deadline first), then completion
	// ASC (least complete first). Filters out completed/closed jobs so the
	// widget surfaces still-running risk, not historical.
	const query = `
		WITH phase_counts AS (
			SELECT
				p.job_id,
				COUNT(*)                                            AS total,
				SUM(CASE WHEN p.status = 'JOB_PHASE_STATUS_COMPLETED' THEN 1 ELSE 0 END) AS done
			FROM job_phase p
			WHERE p.active = true
			GROUP BY p.job_id
		)
		SELECT
			j.id,
			j.name,
			j.status,
			COALESCE(pc.done * 100.0 / NULLIF(pc.total, 0), 0)::float8 AS completion_pct,
			j.planned_end
		FROM job j
		LEFT JOIN phase_counts pc ON pc.job_id = j.id
		WHERE j.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR j.workspace_id = $1)
		  AND j.status NOT IN ('JOB_STATUS_COMPLETED', 'JOB_STATUS_CLOSED')
		  AND j.planned_end IS NOT NULL
		ORDER BY j.planned_end ASC, completion_pct ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
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
