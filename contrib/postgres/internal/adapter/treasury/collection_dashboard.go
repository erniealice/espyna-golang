//go:build postgresql

package treasury

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// SumPending returns the sum of amounts for collection records still in
// "pending" status (centavos). Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_treasury_collection_workspace_status
//	  ON treasury_collection(workspace_id, status) WHERE active = true;
func (r *PostgresCollectionRepository) SumPending(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)::bigint
		FROM treasury_collection tc
		WHERE tc.active = true
		  AND tc.status = 'pending'
		  AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// SumOverdue returns the sum of amounts for collection records that are still
// pending and whose payment_date is before asOf (centavos). Workspace-scoped.
func (r *PostgresCollectionRepository) SumOverdue(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)::bigint
		FROM treasury_collection tc
		WHERE tc.active = true
		  AND tc.status = 'pending'
		  AND tc.payment_date IS NOT NULL
		  AND tc.payment_date < $2
		  AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, asOf).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// SumCollectedToday returns the sum of completed collection amounts whose
// payment_date is on the same calendar day as today (centavos).
// Workspace-scoped.
func (r *PostgresCollectionRepository) SumCollectedToday(
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
		SELECT COALESCE(SUM(tc.amount), 0)::bigint
		FROM treasury_collection tc
		WHERE tc.active = true
		  AND tc.status = 'completed'
		  AND tc.payment_date >= $2
		  AND tc.payment_date < $3
		  AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, dayStart, dayEnd).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// SumByModeWeek groups completed collections in the week starting at weekStart
// by payment_method, returning a map of method → centavos sum. Workspace-scoped.
func (r *PostgresCollectionRepository) SumByModeWeek(
	ctx context.Context,
	workspaceID string,
	weekStart time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	weekEnd := weekStart.AddDate(0, 0, 7)

	const query = `
		SELECT COALESCE(tc.collection_method_id, 'other'), COALESCE(SUM(tc.amount), 0)::bigint
		FROM treasury_collection tc
		WHERE tc.active = true
		  AND tc.status = 'completed'
		  AND tc.payment_date >= $2
		  AND tc.payment_date < $3
		  AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)
		GROUP BY tc.collection_method_id`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, weekStart, weekEnd)
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

// RecentByDate returns the most recent collections newest-first.
// Workspace-scoped.
func (r *PostgresCollectionRepository) RecentByDate(
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

	const query = `
		SELECT to_jsonb(tc) AS row
		FROM treasury_collection tc
		WHERE tc.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)
		ORDER BY COALESCE(tc.payment_date, tc.date_created) DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent collections: %w", err)
	}
	defer rows.Close()

	out := make([]*collectionpb.Collection, 0, limit)
	for rows.Next() {
		var rowJSON []byte
		if scanErr := rows.Scan(&rowJSON); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent collection row: %w", scanErr)
		}

		var rowMap map[string]any
		if err := json.Unmarshal(rowJSON, &rowMap); err != nil {
			log.Printf("WARN: unmarshal recent collection row: %v", err)
			continue
		}
		postgresCore.ConvertMillisToDateStr(rowMap, "payment_date")
		// Re-marshal for protojson decode.
		clean, err := json.Marshal(rowMap)
		if err != nil {
			log.Printf("WARN: re-marshal recent collection row: %v", err)
			continue
		}
		c := &collectionpb.Collection{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(clean, c); err != nil {
			log.Printf("WARN: protojson unmarshal recent collection: %v", err)
			continue
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent collection rows: %w", err)
	}
	return out, nil
}

// SumByDayLast30 returns one TimeBucket per day in the last 30 days ending at
// asOf (inclusive of asOf), with each bucket's value being the sum (centavos)
// of completed collections paid on that day. Workspace-scoped.
func (r *PostgresCollectionRepository) SumByDayLast30(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	end := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())
	start := end.AddDate(0, 0, -29)

	const query = `
		WITH days AS (
			SELECT generate_series(
				date_trunc('day', $2::timestamp),
				date_trunc('day', $3::timestamp),
				interval '1 day'
			) AS bucket
		)
		SELECT d.bucket,
		       COALESCE(SUM(tc.amount), 0)::bigint
		FROM days d
		LEFT JOIN treasury_collection tc
		  ON tc.active = true
		 AND tc.status = 'completed'
		 AND tc.payment_date >= d.bucket
		 AND tc.payment_date < d.bucket + interval '1 day'
		 AND ($1::text IS NULL OR $1::text = '' OR tc.workspace_id = $1)
		GROUP BY d.bucket
		ORDER BY d.bucket ASC`

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
