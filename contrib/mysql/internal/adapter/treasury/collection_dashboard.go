//go:build mysql

package treasury

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// SumPending returns the sum of amounts for collection records still in
// "pending" status (centavos). Workspace-scoped.
//
// Dialect changes from postgres:
//   - $1::text IS NULL OR $1::text = ” → ? IS NULL OR ? = ” (MySQL)
//   - active = true → active = 1
//   - COALESCE(SUM(tc.amount), 0)::bigint → COALESCE(SUM(tc.amount), 0) (MySQL returns DECIMAL for SUM)
func (r *MySQLCollectionRepository) SumPending(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	// Args: workspaceID (null check), workspaceID
	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'pending'
		  AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)`

	var total int64
	if err := mysqlCore.RunDashboardAggregate(ctx, r.db, query, []any{workspaceID, workspaceID, workspaceID}, &total); err != nil {
		return 0, err
	}
	return total, nil
}

// SumOverdue returns the sum of amounts for collection records that are still
// pending and whose payment_date is before asOf (centavos). Workspace-scoped.
func (r *MySQLCollectionRepository) SumOverdue(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	// Args: workspaceID x3 (null check + equality), asOf
	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'pending'
		  AND tc.payment_date IS NOT NULL
		  AND tc.payment_date < ?
		  AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)`

	var total int64
	if err := mysqlCore.RunDashboardAggregate(ctx, r.db, query, []any{asOf, workspaceID, workspaceID, workspaceID}, &total); err != nil {
		return 0, err
	}
	return total, nil
}

// SumCollectedToday returns the sum of completed collection amounts whose
// payment_date is on the same calendar day as today (centavos). Workspace-scoped.
func (r *MySQLCollectionRepository) SumCollectedToday(
	ctx context.Context,
	workspaceID string,
	today time.Time,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	// Args: dayStart, dayEnd, workspaceID x3
	const query = `
		SELECT COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'completed'
		  AND tc.payment_date >= ?
		  AND tc.payment_date < ?
		  AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)`

	var total int64
	if err := mysqlCore.RunDashboardAggregate(ctx, r.db, query, []any{dayStart, dayEnd, workspaceID, workspaceID, workspaceID}, &total); err != nil {
		return 0, err
	}
	return total, nil
}

// SumByModeWeek groups completed collections in the week starting at weekStart
// by payment_method, returning a map of method → centavos sum. Workspace-scoped.
func (r *MySQLCollectionRepository) SumByModeWeek(
	ctx context.Context,
	workspaceID string,
	weekStart time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	weekEnd := weekStart.AddDate(0, 0, 7)

	// Args: weekStart, weekEnd, workspaceID x3
	const query = `
		SELECT COALESCE(tc.collection_method_id, 'other'), COALESCE(SUM(tc.amount), 0)
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND tc.status = 'completed'
		  AND tc.payment_date >= ?
		  AND tc.payment_date < ?
		  AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)
		GROUP BY tc.collection_method_id`

	rows, err := r.db.QueryContext(ctx, query, weekStart, weekEnd, workspaceID, workspaceID, workspaceID)
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
// Dialect changes:
//   - to_jsonb(tc) AS row → explicit column list + manual JSON marshaling
//     (MySQL has no to_jsonb; we select columns explicitly and build the proto)
//   - $1/$2 → ?/?
//   - active = true → active = 1
func (r *MySQLCollectionRepository) RecentByDate(
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

	// Args: workspaceID x3, limit
	const query = `
		SELECT
			tc.id, tc.active, tc.name, tc.amount, tc.status, tc.currency,
			tc.reference_number, tc.payment_date, tc.collection_type,
			tc.subscription_id, tc.revenue_id, tc.collection_method_id,
			tc.received_by, tc.received_role, tc.date_created, tc.date_modified
		FROM treasury_collection tc
		WHERE tc.active = 1
		  AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)
		ORDER BY COALESCE(tc.payment_date, tc.date_created) DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent collections: %w", err)
	}
	defer rows.Close()

	out := make([]*collectionpb.Collection, 0, limit)
	for rows.Next() {
		var (
			id                 string
			active             bool
			name               string
			amount             int64
			status             *string
			currency           *string
			referenceNumber    *string
			paymentDate        *time.Time
			collectionType     *string
			subscriptionID     *string
			revenueID          *string
			collectionMethodID *string
			receivedBy         *string
			receivedRole       *string
			dateCreated        time.Time
			dateModified       time.Time
		)
		if scanErr := rows.Scan(
			&id, &active, &name, &amount, &status, &currency,
			&referenceNumber, &paymentDate, &collectionType,
			&subscriptionID, &revenueID, &collectionMethodID,
			&receivedBy, &receivedRole, &dateCreated, &dateModified,
		); scanErr != nil {
			log.Printf("WARN: scan recent collection row: %v", scanErr)
			continue
		}

		// Build via JSON round-trip to reuse protojson unmarshal path.
		rowMap := map[string]any{
			"id":     id,
			"active": active,
			"name":   name,
			"amount": amount,
		}
		if status != nil {
			rowMap["status"] = *status
		}
		if currency != nil {
			rowMap["currency"] = *currency
		}
		if referenceNumber != nil {
			rowMap["reference_number"] = *referenceNumber
		}
		if paymentDate != nil && !paymentDate.IsZero() {
			rowMap["payment_date"] = paymentDate.Format("2006-01-02")
		}
		if collectionType != nil {
			rowMap["collection_type"] = *collectionType
		}
		if subscriptionID != nil {
			rowMap["subscription_id"] = *subscriptionID
		}
		if revenueID != nil {
			rowMap["revenue_id"] = *revenueID
		}
		if collectionMethodID != nil {
			rowMap["collection_method_id"] = *collectionMethodID
		}
		if receivedBy != nil {
			rowMap["received_by"] = *receivedBy
		}
		if receivedRole != nil {
			rowMap["received_role"] = *receivedRole
		}

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
		if !dateCreated.IsZero() {
			ms := dateCreated.UnixMilli()
			c.DateCreated = &ms
			s := dateCreated.Format(time.RFC3339)
			c.DateCreatedString = &s
		}
		if !dateModified.IsZero() {
			ms := dateModified.UnixMilli()
			c.DateModified = &ms
			s := dateModified.Format(time.RFC3339)
			c.DateModifiedString = &s
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent collection rows: %w", err)
	}
	return out, nil
}

// SumByDayLast30 returns one TimeBucket per day in the last 30 days.
//
// Dialect changes:
//   - generate_series → recursive CTE calendar (MySQL 8.0+)
//   - date_trunc('day', ...) → DATE(...)
//   - interval '1 day' → INTERVAL 1 DAY
//   - $1/$2/$3 → ?/?/?
func (r *MySQLCollectionRepository) SumByDayLast30(
	ctx context.Context,
	workspaceID string,
	asOf time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	end := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())
	start := end.AddDate(0, 0, -29)

	// MySQL 8.0+ recursive CTE to generate a 30-day calendar.
	// Args: start (anchor), end (termination), workspaceID x3
	const query = `
		WITH RECURSIVE days AS (
			SELECT DATE(?) AS bucket
			UNION ALL
			SELECT bucket + INTERVAL 1 DAY FROM days WHERE bucket < DATE(?)
		)
		SELECT d.bucket,
		       COALESCE(SUM(tc.amount), 0)
		FROM days d
		LEFT JOIN treasury_collection tc
		  ON tc.active = 1
		 AND tc.status = 'completed'
		 AND tc.payment_date >= d.bucket
		 AND tc.payment_date < d.bucket + INTERVAL 1 DAY
		 AND (? IS NULL OR ? = '' OR tc.workspace_id = ?)
		GROUP BY d.bucket
		ORDER BY d.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, start, end, workspaceID, workspaceID, workspaceID)
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
