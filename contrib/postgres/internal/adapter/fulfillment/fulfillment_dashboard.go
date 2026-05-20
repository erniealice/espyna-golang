//go:build postgresql

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"

	fulfillmentdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: alias TimeBucket
// from the service-layer fulfillment dashboard package so the adapter's
// named return type matches the interface.
type TimeBucket = fulfillmentdash.TimeBucket

// CountByStatus returns a map of fulfillment.status (canonical string e.g.
// "PENDING", "IN_TRANSIT", "DELIVERED", "EXCEPTION", "CANCELLED") to count,
// restricted to fulfillments created since `since` (when non-zero).
// Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_fulfillment_workspace_status_created
//	  ON fulfillment(workspace_id, status, date_created)
//	  WHERE active = true;
func (r *PostgresFulfillmentRepository) CountByStatus(
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
			SELECT f.status, COUNT(*)::bigint
			FROM fulfillment f
			WHERE f.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR f.workspace_id = $1)
			GROUP BY f.status`
		rows, err = r.db.QueryContext(ctx, q, workspaceID)
	} else {
		const q = `
			SELECT f.status, COUNT(*)::bigint
			FROM fulfillment f
			WHERE f.active = true
			  AND f.date_created >= $2
			  AND ($1::text IS NULL OR $1::text = '' OR f.workspace_id = $1)
			GROUP BY f.status`
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
			return nil, fmt.Errorf("failed to scan fulfillment count row: %w", scanErr)
		}
		key := ""
		if status.Valid {
			key = status.String
		}
		out[key] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment count rows: %w", err)
	}
	return out, nil
}

// AvgFulfillmentTimeDays returns the average elapsed time (in days) between
// fulfillment.date_created and the moment a "DELIVERED" status event was
// recorded for that fulfillment. Restricted to deliveries that occurred on
// or after `since` (when non-zero). Workspace-scoped.
//
// Returns 0 when no qualifying rows exist (rather than NULL/error) so the
// dashboard renders a clean zero value.
func (r *PostgresFulfillmentRepository) AvgFulfillmentTimeDays(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (float64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (e.occurred_at - f.date_created)) / 86400.0), 0)::float8
		FROM fulfillment f
		JOIN fulfillment_status_event e ON e.fulfillment_id = f.id
		WHERE f.active = true
		  AND e.to_status = 'DELIVERED'
		  AND ($1::text IS NULL OR $1::text = '' OR f.workspace_id = $1)
		  AND ($2::timestamptz IS NULL OR e.occurred_at >= $2::timestamptz)`

	var avgDays float64
	var sinceArg interface{}
	if !since.IsZero() {
		sinceArg = since
	}
	if err := r.db.QueryRowContext(ctx, query, workspaceID, sinceArg).Scan(&avgDays); err != nil {
		return 0, nil //nolint:nilerr
	}
	return avgDays, nil
}

// RecentExceptions returns the most recent fulfillments whose status is
// 'EXCEPTION' or 'CANCELLED' (the operator-actionable failure surface),
// ordered by date_modified DESC. Workspace-scoped.
//
// The schema has no canonical "EXCEPTION" status today — the canonical
// failure-class statuses are FAILED and CANCELLED (see fulfillment.proto
// status comment: "PENDING, READY, IN_TRANSIT, DELIVERED, PARTIALLY_DELIVERED,
// FAILED, CANCELLED"). We treat both FAILED and CANCELLED as exceptions for
// the dashboard widget.
func (r *PostgresFulfillmentRepository) RecentExceptions(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*pb.Fulfillment, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			f.id,
			f.status,
			f.delivery_mode,
			f.provider_status,
			f.provider_reference,
			f.date_created,
			f.date_modified
		FROM fulfillment f
		WHERE f.active = true
		  AND f.status IN ('FAILED', 'CANCELLED', 'EXCEPTION')
		  AND ($1::text IS NULL OR $1::text = '' OR f.workspace_id = $1)
		ORDER BY f.date_modified DESC NULLS LAST
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]*pb.Fulfillment, 0, limit)
	for rows.Next() {
		var (
			id             string
			status         sql.NullString
			deliveryMode   sql.NullString
			providerStatus sql.NullString
			providerRef    sql.NullString
			dateCreated    sql.NullTime
			dateModified   sql.NullTime
		)
		if scanErr := rows.Scan(
			&id, &status, &deliveryMode, &providerStatus, &providerRef,
			&dateCreated, &dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan fulfillment exception row: %w", scanErr)
		}
		f := &pb.Fulfillment{Id: id, Active: true}
		if status.Valid {
			f.Status = status.String
		}
		if deliveryMode.Valid {
			f.DeliveryMode = deliveryMode.String
		}
		if providerStatus.Valid {
			f.ProviderStatus = providerStatus.String
		}
		if providerRef.Valid {
			f.ProviderReference = providerRef.String
		}
		if dateCreated.Valid {
			ms := dateCreated.Time.UnixMilli()
			f.DateCreated = &ms
		}
		if dateModified.Valid {
			ms := dateModified.Time.UnixMilli()
			f.DateModified = &ms
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment exception rows: %w", err)
	}
	return out, nil
}

// DailyDeliveredLast30 returns one TimeBucket per day in the trailing 30 days
// ending at `asOf` (defaulting to NOW()), counting fulfillment_status_event
// rows whose to_status = 'DELIVERED' on that day. Workspace-scoped via the
// joined fulfillment row's workspace_id.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_fulfillment_status_event_occurred_status
//	  ON fulfillment_status_event(to_status, occurred_at);
func (r *PostgresFulfillmentRepository) DailyDeliveredLast30(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}

	// codex-review-phase1-round2b P0 fix (2026-05-21): count f.id (not e.id) so
	// that the workspace LEFT JOIN actually scopes the aggregate. Counting e.id
	// would include event rows that join to no fulfillment row (i.e. wrong
	// workspace), inflating the delivered count across workspaces. With
	// COUNT(f.id) the workspace predicate on the fulfillment join effectively
	// becomes a workspace filter on what gets counted.
	const query = `
		WITH days AS (
			SELECT generate_series(
				date_trunc('day', $2::timestamptz - interval '29 days'),
				date_trunc('day', $2::timestamptz),
				interval '1 day'
			) AS bucket
		)
		SELECT
			d.bucket,
			COUNT(f.id)::bigint AS delivered
		FROM days d
		LEFT JOIN fulfillment_status_event e
			ON date_trunc('day', e.occurred_at) = d.bucket
			AND e.to_status = 'DELIVERED'
		LEFT JOIN fulfillment f
			ON f.id = e.fulfillment_id
			AND ($1::text IS NULL OR $1::text = '' OR f.workspace_id = $1)
		GROUP BY d.bucket
		ORDER BY d.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, asOf)
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
			return nil, fmt.Errorf("failed to scan daily-delivered row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily-delivered rows: %w", err)
	}
	return out, nil
}
