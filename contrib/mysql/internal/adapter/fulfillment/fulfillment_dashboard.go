//go:build mysql

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"

	fulfillmentdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
)

// TimeBucket is a named-type alias from the service-layer fulfillment dashboard package
// so the adapter's named return type matches the interface.
type TimeBucket = fulfillmentdash.TimeBucket

// CountByStatus returns a map of fulfillment.status to count.
// Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - COUNT(*)::bigint → COUNT(*) (MySQL returns int64 natively)
//   - $1::text IS NULL OR ... → (? = ” OR f.workspace_id = ?)
//   - $2 for since date → ?
//   - active = true → active = 1
func (r *MySQLFulfillmentRepository) CountByStatus(
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
			SELECT f.status, COUNT(*)
			FROM fulfillment f
			WHERE f.active = 1
			  AND (? = '' OR f.workspace_id = ?)
			GROUP BY f.status`
		rows, err = r.db.QueryContext(ctx, q, workspaceID, workspaceID)
	} else {
		const q = `
			SELECT f.status, COUNT(*)
			FROM fulfillment f
			WHERE f.active = 1
			  AND f.date_created >= ?
			  AND (? = '' OR f.workspace_id = ?)
			GROUP BY f.status`
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
// recorded. Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - EXTRACT(EPOCH FROM (e.occurred_at - f.date_created)) / 86400.0 →
//     TIMESTAMPDIFF(SECOND, f.date_created, e.occurred_at) / 86400.0
//   - COALESCE(AVG(...), 0)::float8 → COALESCE(AVG(...), 0)
//   - $2::timestamptz IS NULL OR ... → (? = ” OR e.occurred_at >= ?)
//   - active = true → active = 1
func (r *MySQLFulfillmentRepository) AvgFulfillmentTimeDays(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (float64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	// Dialect: EXTRACT(EPOCH FROM (... - ...)) → TIMESTAMPDIFF(SECOND, ..., ...)
	const query = `
		SELECT COALESCE(AVG(TIMESTAMPDIFF(SECOND, f.date_created, e.occurred_at) / 86400.0), 0)
		FROM fulfillment f
		JOIN fulfillment_status_event e ON e.fulfillment_id = f.id
		WHERE f.active = 1
		  AND e.to_status = 'DELIVERED'
		  AND (? = '' OR f.workspace_id = ?)
		  AND (? = '' OR e.occurred_at >= ?)`

	sinceStr := ""
	if !since.IsZero() {
		sinceStr = since.Format("2006-01-02 15:04:05")
	}

	var avgDays float64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, workspaceID, sinceStr, sinceStr).Scan(&avgDays); err != nil {
		return 0, nil //nolint:nilerr
	}
	return avgDays, nil
}

// RecentExceptions returns the most recent fulfillments whose status is
// 'FAILED', 'CANCELLED', or 'EXCEPTION'. Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - $1::text IS NULL OR ... → (? = ” OR f.workspace_id = ?)
//   - ORDER BY ... NULLS LAST → ORDER BY ... IS NULL ASC, ... DESC
//   - LIMIT $2 → LIMIT ?
//   - active = true → active = 1
func (r *MySQLFulfillmentRepository) RecentExceptions(
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

	// Dialect: NULLS LAST → IS NULL ASC (MySQL, for DESC ordering)
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
		WHERE f.active = 1
		  AND f.status IN ('FAILED', 'CANCELLED', 'EXCEPTION')
		  AND (? = '' OR f.workspace_id = ?)
		ORDER BY f.date_modified IS NULL ASC, f.date_modified DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, limit)
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
// ending at `asOf`, counting fulfillment_status_event rows whose to_status =
// 'DELIVERED' on that day. Workspace-scoped.
//
// Dialect translation from postgres gold standard:
//   - generate_series() → recursive CTE (MySQL has no generate_series)
//   - date_trunc('day', ...) → DATE(...)
//   - COUNT(f.id)::bigint → COUNT(f.id)
//   - $1::text IS NULL OR ... → (? = ” OR f.workspace_id = ?)
//   - $2::timestamptz → ?
//   - active = true → active = 1
func (r *MySQLFulfillmentRepository) DailyDeliveredLast30(
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

	// MySQL recursive CTE to generate 30 day buckets ending at asOf.
	// DATE_SUB(?, INTERVAL 29 DAY) gives the window start (30 days inclusive).
	const query = `
		WITH RECURSIVE days AS (
			SELECT DATE(?) AS bucket, 0 AS n
			UNION ALL
			SELECT DATE_SUB(bucket, INTERVAL 1 DAY), n + 1
			FROM days
			WHERE n < 29
		)
		SELECT
			d.bucket,
			COUNT(f.id) AS delivered
		FROM days d
		LEFT JOIN fulfillment_status_event e
			ON DATE(e.occurred_at) = d.bucket
			AND e.to_status = 'DELIVERED'
		LEFT JOIN fulfillment f
			ON f.id = e.fulfillment_id
			AND (? = '' OR f.workspace_id = ?)
		GROUP BY d.bucket
		ORDER BY d.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, asOf, workspaceID, workspaceID)
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
