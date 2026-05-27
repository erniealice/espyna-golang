//go:build sqlserver

package treasury

import (
	"context"
	"fmt"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// TimeBucket is aliased to the service-layer Go-only TimeBucket so the
// adapter's dashboard methods satisfy the service-layer interfaces exactly.
// (Mirrors the same aliasing pattern used in the postgres gold standard.)
type TimeBucket = treasurydash.TimeBucket

// SumPending returns the sum of amounts for collection records in "pending"
// status (centavos). Workspace-scoped.
//
// SQL Server translation:
//   - @p1 instead of $1.
//   - No ::text cast — @p1 is natively a string parameter; empty-string check
//     using (@p1 = ” OR tc.workspace_id = @p1).
//   - active = 1 (BIT) instead of active = true.
//   - RunDashboardAggregate uses SUM(CASE WHEN …) internally — not needed here
//     (single aggregate, no FILTER clause), but we route through
//     RunDashboardAggregate for the honest error seam (A5 fix).
func (r *SQLServerCollectionRepository) SumPending(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'pending'
		  AND (@p1 = '' OR tc.workspace_id = @p1)`

	var total int64
	if err := sqlserverCore.RunDashboardAggregate(
		ctx, r.db, query, []any{workspaceID}, &total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// SumOverdue returns the sum of amounts for pending collections whose
// payment_date is before asOf (centavos). Workspace-scoped.
func (r *SQLServerCollectionRepository) SumOverdue(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'pending'
		  AND tc.payment_date IS NOT NULL
		  AND tc.payment_date < @p2
		  AND (@p1 = '' OR tc.workspace_id = @p1)`

	var total int64
	if err := sqlserverCore.RunDashboardAggregate(
		ctx, r.db, query, []any{workspaceID, asOf}, &total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// SumCollectedToday returns the sum of completed collection amounts whose
// payment_date is on the same calendar day as today (centavos). Workspace-scoped.
func (r *SQLServerCollectionRepository) SumCollectedToday(
	ctx context.Context,
	workspaceID string,
	today time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'completed'
		  AND tc.payment_date >= @p2
		  AND tc.payment_date < @p3
		  AND (@p1 = '' OR tc.workspace_id = @p1)`

	var total int64
	if err := sqlserverCore.RunDashboardAggregate(
		ctx, r.db, query, []any{workspaceID, dayStart, dayEnd}, &total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// SumByModeWeek groups completed collections in the week starting at weekStart
// by payment_method, returning a map of method → centavos sum. Workspace-scoped.
func (r *SQLServerCollectionRepository) SumByModeWeek(
	ctx context.Context,
	workspaceID string,
	weekStart time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	weekEnd := weekStart.AddDate(0, 0, 7)

	// SQL Server: COALESCE for the nullable collection_method_id grouping key;
	// no generate_series needed — just the direct aggregate.
	const query = `
		SELECT COALESCE(tc.collection_method_id, 'other'), COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'completed'
		  AND tc.payment_date >= @p2
		  AND tc.payment_date < @p3
		  AND (@p1 = '' OR tc.workspace_id = @p1)
		GROUP BY tc.collection_method_id`

	exec := r.db
	rows, err := exec.QueryContext(ctx, query, workspaceID, weekStart, weekEnd)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 6)
	for rows.Next() {
		var (
			mode string
			sum  int64
		)
		if scanErr := rows.Scan(&mode, &sum); scanErr != nil {
			return nil, fmt.Errorf("failed to scan collection-by-mode row: %w", scanErr)
		}
		out[mode] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection-by-mode rows: %w", err)
	}
	return out, nil
}

// RecentByDate returns the most recent collections newest-first. Workspace-scoped.
//
// SQL Server: explicit column list instead of to_jsonb(); TOP @p2 instead of
// LIMIT $2; manual proto construction instead of JSON round-trip.
func (r *SQLServerCollectionRepository) RecentByDate(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*collectionpb.Collection, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		SELECT TOP %d
			tc.id, tc.active, tc.name, tc.amount, tc.status,
			tc.currency, tc.reference_number, tc.payment_date, tc.collection_type
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND (@p1 = '' OR tc.workspace_id = @p1)
		ORDER BY COALESCE(tc.payment_date, tc.date_created) DESC`, limit)

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent collections: %w", err)
	}
	defer rows.Close()

	out := make([]*collectionpb.Collection, 0, limit)
	for rows.Next() {
		c := &collectionpb.Collection{}
		var paymentDate *string
		if scanErr := rows.Scan(
			&c.Id, &c.Active, &c.Name, &c.Amount, &c.Status,
			&c.Currency, &c.ReferenceNumber, &paymentDate, &c.CollectionType,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent collection row: %w", scanErr)
		}
		if paymentDate != nil {
			c.PaymentDate = *paymentDate
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent collection rows: %w", err)
	}
	return out, nil
}

// SumByDayLast30 returns one TimeBucket per day in the last 30 days ending at
// asOf (inclusive), with each bucket's value being the sum (centavos) of
// completed collections paid on that day. Workspace-scoped.
//
// SQL Server: uses a recursive CTE to generate the 30-day date series
// (SQL Server has no generate_series). OUTER APPLY replaces the LEFT JOIN
// LATERAL pattern. One row per day even if there are no collections.
func (r *SQLServerCollectionRepository) SumByDayLast30(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	end := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())
	start := end.AddDate(0, 0, -29)

	// Recursive CTE generates one row per day; OUTER APPLY aggregates matching
	// collection rows for each day.
	const query = `
		WITH days AS (
			SELECT CAST(@p2 AS datetime2) AS bucket
			UNION ALL
			SELECT DATEADD(day, 1, bucket) FROM days
			WHERE bucket < CAST(@p3 AS datetime2)
		)
		SELECT d.bucket,
		       COALESCE(agg.day_sum, 0) AS day_sum
		FROM days d
		OUTER APPLY (
			SELECT SUM(tc.amount) AS day_sum
			FROM treasury_collection tc
			WHERE tc.active = 1
			  AND tc.status = 'completed'
			  AND tc.payment_date >= d.bucket
			  AND tc.payment_date < DATEADD(day, 1, d.bucket)
			  AND (@p1 = '' OR tc.workspace_id = @p1)
		) agg
		ORDER BY d.bucket ASC
		OPTION (MAXRECURSION 35)`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, start, end)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]TimeBucket, 0, 30)
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan collection-by-day row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection-by-day rows: %w", err)
	}
	return out, nil
}
