//go:build postgresql

package event

import (
	"context"
	"fmt"
	"time"

	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	"github.com/erniealice/espyna-golang/database/operations"
)

// TimeBucket is a generic (period, value) tuple for time-series aggregates,
// shared across event dashboard methods.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// CountToday returns the number of events whose start_date_time_utc falls on
// the given UTC date. Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_event_workspace_start
//	  ON event(workspace_id, start_date_time_utc)
//	  WHERE active = true;
func (r *PostgresEventRepository) CountToday(
	ctx context.Context,
	workspaceID string,
	today time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	const query = `
		SELECT COUNT(*)::bigint
		FROM event e
		WHERE e.active = true
		  AND e.start_date_time_utc >= $2
		  AND e.start_date_time_utc < $3
		  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)`

	var n int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, dayStart.UnixMilli(), dayEnd.UnixMilli()).Scan(&n); err != nil {
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

// CountThisWeek returns the number of events whose start_date_time_utc falls
// within the 7-day window starting at weekStart. Workspace-scoped.
func (r *PostgresEventRepository) CountThisWeek(
	ctx context.Context,
	workspaceID string,
	weekStart time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	from := time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)

	const query = `
		SELECT COUNT(*)::bigint
		FROM event e
		WHERE e.active = true
		  AND e.start_date_time_utc >= $2
		  AND e.start_date_time_utc < $3
		  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)`

	var n int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, from.UnixMilli(), to.UnixMilli()).Scan(&n); err != nil {
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

// UpcomingByStartDate returns active events with start_date_time_utc >= now,
// ordered by start_date_time_utc ASC. Workspace-scoped.
func (r *PostgresEventRepository) UpcomingByStartDate(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*eventpb.Event, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	nowMillis := time.Now().UTC().UnixMilli()

	const query = `
		SELECT
			e.id,
			e.name,
			e.description,
			e.start_date_time_utc,
			e.end_date_time_utc,
			e.active,
			e.date_created,
			e.date_modified
		FROM event e
		WHERE e.active = true
		  AND e.start_date_time_utc >= $2
		  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)
		ORDER BY e.start_date_time_utc ASC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, nowMillis, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
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
func (r *PostgresEventRepository) CountByDay(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if from.After(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	toEnd := toDay.Add(24 * time.Hour)

	const query = `
		WITH days AS (
			SELECT generate_series(
				$2::timestamp,
				$3::timestamp,
				interval '1 day'
			)::date AS bucket
		),
		event_days AS (
			SELECT to_timestamp(e.start_date_time_utc / 1000)::date AS bucket,
			       COUNT(*)::bigint AS n
			FROM event e
			WHERE e.active = true
			  AND e.start_date_time_utc >= $4
			  AND e.start_date_time_utc < $5
			  AND ($1::text IS NULL OR $1::text = '' OR e.workspace_id = $1)
			GROUP BY 1
		)
		SELECT d.bucket, COALESCE(ed.n, 0)::bigint
		FROM days d
		LEFT JOIN event_days ed ON ed.bucket = d.bucket
		ORDER BY d.bucket ASC`

	rows, err := r.db.QueryContext(
		ctx, query, workspaceID,
		fromDay, toDay,
		fromDay.UnixMilli(), toEnd.UnixMilli(),
	)
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
			return nil, fmt.Errorf("failed to scan event count-by-day row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event count-by-day rows: %w", err)
	}
	return out, nil
}

// CountByTag returns a map of tag-name to count, joining event_tag_assignment
// + event_tag and counting distinct active events per tag. Workspace-scoped on
// event_tag (event_tag_assignment has no workspace column in the schema; the
// event_tag join enforces tenant scope).
func (r *PostgresEventRepository) CountByTag(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT et.name, COUNT(DISTINCT eta.event_id)::bigint
		FROM event_tag_assignment eta
		JOIN event_tag et ON et.id = eta.event_tag_id
		WHERE eta.active = true
		  AND et.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR et.workspace_id = $1)
		GROUP BY et.name
		ORDER BY 2 DESC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
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
