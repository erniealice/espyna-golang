//go:build sqlserver

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

// CountByStatus returns a map of fulfillment.status to count,
// restricted to fulfillments created since `since` (when non-zero). Workspace-scoped.
//
// SQL Server differences from postgres gold standard:
//   - $N → @pN.
//   - active = true → active = 1.
//   - ::bigint → removed (SQL Server infers from COUNT).
//   - ::text cast for workspace_id check → (@p1 IS NULL OR @p1 = ”).
func (r *SQLServerFulfillmentRepository) CountByStatus(
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
			SELECT f.status, COUNT(*)
			FROM fulfillment f
			WHERE f.active = 1
			  AND (@p1 IS NULL OR @p1 = '' OR f.workspace_id = @p1)
			GROUP BY f.status`
		rows, queryErr = exec.QueryContext(ctx, q, workspaceID)
	} else {
		const q = `
			SELECT f.status, COUNT(*)
			FROM fulfillment f
			WHERE f.active = 1
			  AND f.date_created >= @p2
			  AND (@p1 IS NULL OR @p1 = '' OR f.workspace_id = @p1)
			GROUP BY f.status`
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
// fulfillment.date_created and the moment a "DELIVERED" status event was recorded.
//
// SQL Server differences from postgres gold standard:
//   - EXTRACT(EPOCH FROM ...) / 86400.0 → DATEDIFF(second, f.date_created, e.occurred_at) / 86400.0.
//   - ::timestamptz → removed; @p2 is compared directly.
//   - COALESCE(..., 0)::float8 → CAST(COALESCE(..., 0) AS float).
func (r *SQLServerFulfillmentRepository) AvgFulfillmentTimeDays(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (float64, error) {
	const query = `
		SELECT CAST(COALESCE(AVG(CAST(DATEDIFF(second, f.date_created, e.occurred_at) AS float) / 86400.0), 0) AS float)
		FROM fulfillment f
		JOIN fulfillment_status_event e ON e.fulfillment_id = f.id
		WHERE f.active = 1
		  AND e.to_status = 'DELIVERED'
		  AND (@p1 IS NULL OR @p1 = '' OR f.workspace_id = @p1)
		  AND (@p2 IS NULL OR e.occurred_at >= @p2)`

	var avgDays float64
	var sinceArg interface{}
	if !since.IsZero() {
		sinceArg = since
	}
	exec := r.getExec(ctx)
	if err := exec.QueryRowContext(ctx, query, workspaceID, sinceArg).Scan(&avgDays); err != nil {
		return 0, nil //nolint:nilerr
	}
	return avgDays, nil
}

// RecentExceptions returns the most recent fulfillments with failure-class statuses.
//
// SQL Server differences:
//   - active = true → active = 1.
//   - NULLS LAST → CASE WHEN date_modified IS NULL THEN 1 ELSE 0 END (push NULLs last).
//   - LIMIT → TOP.
//   - $N → @pN.
func (r *SQLServerFulfillmentRepository) RecentExceptions(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*pb.Fulfillment, error) {
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		SELECT TOP %d
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
		  AND (@p1 IS NULL OR @p1 = '' OR f.workspace_id = @p1)
		ORDER BY
			CASE WHEN f.date_modified IS NULL THEN 1 ELSE 0 END,
			f.date_modified DESC`, limit)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
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
// counting fulfillment_status_event rows whose to_status = 'DELIVERED' on that day.
//
// SQL Server differences from postgres gold standard:
//   - generate_series → recursive CTE (DATEADD approach; MAXRECURSION 30).
//   - date_trunc('day', ...) → CAST(... AS date).
//   - COUNT(f.id)::bigint → CAST(COUNT(f.id) AS bigint).
//   - $N → @pN.
//   - interval '29 days' → DATEADD(day, -29, CAST(@p2 AS date)).
func (r *SQLServerFulfillmentRepository) DailyDeliveredLast30(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) ([]TimeBucket, error) {
	if asOf.IsZero() {
		asOf = time.Now()
	}

	const query = `
		WITH days AS (
			SELECT CAST(DATEADD(day, -29, CAST(@p2 AS date)) AS datetime2) AS bucket
			UNION ALL
			SELECT DATEADD(day, 1, bucket)
			FROM days
			WHERE DATEADD(day, 1, bucket) <= CAST(@p2 AS date)
		)
		SELECT
			d.bucket,
			CAST(COUNT(f.id) AS bigint) AS delivered
		FROM days d
		LEFT JOIN fulfillment_status_event e
			ON CAST(e.occurred_at AS date) = CAST(d.bucket AS date)
			AND e.to_status = 'DELIVERED'
		LEFT JOIN fulfillment f
			ON f.id = e.fulfillment_id
			AND (@p1 IS NULL OR @p1 = '' OR f.workspace_id = @p1)
		GROUP BY d.bucket
		ORDER BY d.bucket ASC
		OPTION (MAXRECURSION 30)`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, asOf)
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
