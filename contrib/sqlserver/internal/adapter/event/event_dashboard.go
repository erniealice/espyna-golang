//go:build sqlserver

package event

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/database/operations"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// TimeBucket is a generic (period, value) tuple for time-series aggregates,
// shared across event dashboard methods.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// countEventsInWindow runs a scalar count of active events whose
// start_date_time_utc falls in the half-open [startMillis, endMillis) window.
// Workspace-scoped.
//
// SQL Server differences from postgres gold standard:
//   - COUNT(*) FILTER (WHERE ...) → SUM(CASE WHEN ... THEN 1 ELSE 0 END) (A8 rule).
//   - ::text, ::bigint casts → removed.
//   - $N → @pN.
func (r *SQLServerEventRepository) countEventsInWindow(
	ctx context.Context,
	workspaceID string,
	startMillis, endMillis int64,
) (int64, error) {
	const query = `
		WITH base AS (
			SELECT e.start_date_time_utc
			FROM event e
			WHERE e.active = 1
			  AND (@p1 IS NULL OR @p1 = '' OR e.workspace_id = @p1)
		)
		SELECT SUM(CASE WHEN start_date_time_utc >= @p2 AND start_date_time_utc < @p3 THEN 1 ELSE 0 END)
		FROM base`

	exec := r.getExec(ctx)
	var n sql.NullInt64
	if err := exec.QueryRowContext(ctx, query, workspaceID, startMillis, endMillis).Scan(&n); err != nil {
		return 0, nil //nolint:nilerr
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

// CountToday returns the number of events whose start_date_time_utc falls on
// the given UTC date. Workspace-scoped.
func (r *SQLServerEventRepository) CountToday(
	ctx context.Context,
	workspaceID string,
	today time.Time,
) (int64, error) {
	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	return r.countEventsInWindow(ctx, workspaceID, dayStart.UnixMilli(), dayEnd.UnixMilli())
}

// CountThisWeek returns the number of events whose start_date_time_utc falls
// within the 7-day window starting at weekStart. Workspace-scoped.
func (r *SQLServerEventRepository) CountThisWeek(
	ctx context.Context,
	workspaceID string,
	weekStart time.Time,
) (int64, error) {
	from := time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)
	return r.countEventsInWindow(ctx, workspaceID, from.UnixMilli(), to.UnixMilli())
}

// UpcomingByStartDate returns active events with start_date_time_utc >= now,
// ordered by start_date_time_utc ASC. Workspace-scoped.
//
// SQL Server differences:
//   - active = true → active = 1.
//   - $N → @pN.
//   - LIMIT → TOP.
func (r *SQLServerEventRepository) UpcomingByStartDate(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*eventpb.Event, error) {
	if limit <= 0 {
		limit = 5
	}

	nowMillis := time.Now().UTC().UnixMilli()

	query := fmt.Sprintf(`
		SELECT TOP %d
			e.id,
			e.name,
			e.description,
			e.start_date_time_utc,
			e.end_date_time_utc,
			e.active,
			e.date_created,
			e.date_modified
		FROM event e
		WHERE e.active = 1
		  AND e.start_date_time_utc >= @p2
		  AND (@p1 IS NULL OR @p1 = '' OR e.workspace_id = @p1)
		ORDER BY e.start_date_time_utc ASC`, limit)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, nowMillis)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming events: %w", err)
	}
	defer rows.Close()

	out := make([]*eventpb.Event, 0, limit)
	for rows.Next() {
		var (
			id               string
			name             string
			description      *string
			startDateTimeUTC *string
			endDateTimeUTC   *string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
		)
		if scanErr := rows.Scan(
			&id, &name, &description,
			&startDateTimeUTC, &endDateTimeUTC,
			&active, &dateCreated, &dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan upcoming event row: %w", scanErr)
		}
		ev := &eventpb.Event{Id: id, Name: name, Active: active}
		if description != nil {
			ev.Description = description
		}
		if startDateTimeUTC != nil && *startDateTimeUTC != "" {
			if ts, err := operations.ParseTimestamp(*startDateTimeUTC); err == nil {
				ev.StartDateTimeUtc = ts
			}
		}
		if endDateTimeUTC != nil && *endDateTimeUTC != "" {
			if ts, err := operations.ParseTimestamp(*endDateTimeUTC); err == nil {
				ev.EndDateTimeUtc = ts
			}
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			ev.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			ev.DateModified = &ts
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating upcoming event rows: %w", err)
	}
	return out, nil
}

// CountByDay returns one TimeBucket per day in [from, to], counting active
// events whose start_date_time_utc falls inside that day. Workspace-scoped.
//
// SQL Server differences from postgres gold standard:
//   - generate_series → recursive CTE (MAXRECURSION via OPTION clause).
//   - to_timestamp(millis / 1000)::date → CAST(DATEADD(millisecond, millis % 1000, DATEADD(second, millis / 1000, '19700101')) AS date).
//   - $N → @pN.
//   - ::bigint → CAST(... AS bigint).
func (r *SQLServerEventRepository) CountByDay(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) ([]TimeBucket, error) {
	if from.After(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	toEnd := toDay.Add(24 * time.Hour)

	const query = `
		WITH days AS (
			SELECT CAST(@p2 AS date) AS bucket
			UNION ALL
			SELECT DATEADD(day, 1, bucket)
			FROM days
			WHERE DATEADD(day, 1, bucket) <= @p3
		),
		event_days AS (
			SELECT
				CAST(DATEADD(millisecond, e.start_date_time_utc % 1000,
					DATEADD(second, e.start_date_time_utc / 1000,
					CAST('19700101' AS datetime2))) AS date) AS bucket,
				CAST(COUNT(*) AS bigint) AS n
			FROM event e
			WHERE e.active = 1
			  AND e.start_date_time_utc >= @p4
			  AND e.start_date_time_utc < @p5
			  AND (@p1 IS NULL OR @p1 = '' OR e.workspace_id = @p1)
			GROUP BY CAST(DATEADD(millisecond, e.start_date_time_utc % 1000,
				DATEADD(second, e.start_date_time_utc / 1000,
				CAST('19700101' AS datetime2))) AS date)
		)
		SELECT d.bucket, COALESCE(ed.n, 0)
		FROM days d
		LEFT JOIN event_days ed ON ed.bucket = d.bucket
		ORDER BY d.bucket ASC
		OPTION (MAXRECURSION 366)`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(
		ctx, query, workspaceID,
		fromDay, toDay,
		fromDay.UnixMilli(), toEnd.UnixMilli(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query event count-by-day: %w", err)
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan event count-by-day row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event count-by-day rows: %w", err)
	}
	return out, nil
}

// CountByTag returns a map of tag-name to count. Workspace-scoped via event_tag.workspace_id.
//
// SQL Server differences:
//   - active = true → active = 1.
//   - COUNT(DISTINCT ...)::bigint → CAST(COUNT(DISTINCT ...) AS bigint).
//   - $N → @pN.
func (r *SQLServerEventRepository) CountByTag(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	const query = `
		SELECT et.name, CAST(COUNT(DISTINCT eta.event_id) AS bigint)
		FROM event_tag_assignment eta
		JOIN event_tag et ON et.id = eta.event_tag_id
		WHERE eta.active = 1
		  AND et.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR et.workspace_id = @p1)
		GROUP BY et.name
		ORDER BY 2 DESC`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event count-by-tag: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var (
			name string
			n    int64
		)
		if scanErr := rows.Scan(&name, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan event count-by-tag row: %w", scanErr)
		}
		out[name] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event count-by-tag rows: %w", err)
	}
	return out, nil
}
