//go:build sqlserver

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
// `var _ jobdash.JobDashboardRepository = (*SQLServerJobRepository)(nil)`
// succeed in job_dashboard_assertions.go.
type JobRisk = jobdash.JobRisk

// CountByStatus returns a map of job.status (raw enum string) to count,
// restricted to jobs created since `since` (when non-zero). Workspace-scoped.
//
// SQL Server differences from postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1  (BIT column).
//   - ::text / ::bigint casts removed — SQL Server infers from context.
//   - CAST(NULL AS varchar) used for nullable @p1 equivalence check.
func (r *SQLServerJobRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (map[string]int64, error) {
	exec := r.getExec(ctx)

	var (
		rows interface {
			Next() bool
			Scan(...any) error
			Err() error
			Close() error
		}
		queryErr error
	)

	if since.IsZero() {
		const q = `
			SELECT j.status, COUNT(*)
			FROM job j
			WHERE j.active = 1
			  AND (@p1 IS NULL OR @p1 = '' OR j.workspace_id = @p1)
			GROUP BY j.status`
		rows, queryErr = exec.QueryContext(ctx, q, workspaceID)
	} else {
		const q = `
			SELECT j.status, COUNT(*)
			FROM job j
			WHERE j.active = 1
			  AND j.date_created >= @p2
			  AND (@p1 IS NULL OR @p1 = '' OR j.workspace_id = @p1)
			GROUP BY j.status`
		rows, queryErr = exec.QueryContext(ctx, q, workspaceID, since)
	}
	if queryErr != nil {
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
// SQL Server differences from postgres gold standard:
//   - INTERVAL / interval → DATEADD(day, N, GETUTCDATE()).
//   - to_timestamp(due_date / 1000) → DATEADD(millisecond, due_date % 1000, DATEADD(second, due_date / 1000, '19700101')).
//   - $N → @pN.
//   - LIMIT → TOP (applied on the outer SELECT).
//   - NULLS LAST removed (SQL Server puts NULLs last in ASC by default).
func (r *SQLServerJobRepository) UpcomingDeadlines(
	ctx context.Context,
	workspaceID string,
	days int,
	limit int32,
) ([]*jobpb.Job, error) {
	if days <= 0 {
		days = 14
	}
	if limit <= 0 {
		limit = 5
	}

	// SQL Server does not have to_timestamp(); convert epoch-millis due_date via DATEADD.
	// COALESCE(planned_end, <due_date_as_timestamp>) IS NOT NULL and within window.
	query := fmt.Sprintf(`
		SELECT TOP %d
			j.id,
			j.name,
			j.status,
			j.planned_end,
			j.due_date
		FROM job j
		WHERE j.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR j.workspace_id = @p1)
		  AND COALESCE(
			  j.planned_end,
			  DATEADD(millisecond, j.due_date %% 1000,
				DATEADD(second, j.due_date / 1000,
				  CAST('19700101' AS datetime2)))
			) IS NOT NULL
		  AND COALESCE(
			  j.planned_end,
			  DATEADD(millisecond, j.due_date %% 1000,
				DATEADD(second, j.due_date / 1000,
				  CAST('19700101' AS datetime2)))
			) >= GETUTCDATE()
		  AND COALESCE(
			  j.planned_end,
			  DATEADD(millisecond, j.due_date %% 1000,
				DATEADD(second, j.due_date / 1000,
				  CAST('19700101' AS datetime2)))
			) <= DATEADD(day, @p2, GETUTCDATE())
		ORDER BY COALESCE(
			j.planned_end,
			DATEADD(millisecond, j.due_date %% 1000,
				DATEADD(second, j.due_date / 1000,
				  CAST('19700101' AS datetime2)))
		) ASC`, limit)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, days)
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
		_ = status
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating upcoming-deadline rows: %w", err)
	}
	return out, nil
}

// TopByCompletionRisk ranks active, not-yet-completed jobs by completion risk.
//
// SQL Server differences from postgres gold standard:
//   - NULLIF → NULLIF (supported).
//   - ::float8 → CAST(... AS float).
//   - $N → @pN.
//   - LIMIT → TOP.
//   - FILTER (WHERE ...) → CASE WHEN ... END (A8 dialect rule).
func (r *SQLServerJobRepository) TopByCompletionRisk(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]JobRisk, error) {
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		WITH phase_counts AS (
			SELECT
				p.job_id,
				COUNT(*) AS total,
				SUM(CASE WHEN p.status = 'JOB_PHASE_STATUS_COMPLETED' THEN 1 ELSE 0 END) AS done
			FROM job_phase p
			WHERE p.active = 1
			GROUP BY p.job_id
		)
		SELECT TOP %d
			j.id,
			j.name,
			j.status,
			COALESCE(CAST(pc.done AS float) * 100.0 / NULLIF(CAST(pc.total AS float), 0), 0) AS completion_pct,
			j.planned_end
		FROM job j
		LEFT JOIN phase_counts pc ON pc.job_id = j.id
		WHERE j.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR j.workspace_id = @p1)
		  AND j.status NOT IN ('JOB_STATUS_COMPLETED', 'JOB_STATUS_CLOSED')
		  AND j.planned_end IS NOT NULL
		ORDER BY j.planned_end ASC, completion_pct ASC`, limit)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
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
			Code:          name,
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
