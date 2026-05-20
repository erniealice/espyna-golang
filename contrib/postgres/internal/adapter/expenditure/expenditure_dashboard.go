//go:build postgresql

package expenditure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"

	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: the postgres adapter
// MUST return EXACTLY the named row types declared by the service-layer
// dashboard package. Aliasing here keeps signatures identical so the
// compile-time assertion in expenditure_dashboard_assertions.go succeeds.
type (
	TimeBucket     = expendituredash.TimeBucket
	TopSupplierRow = expendituredash.TopSupplierRow
)

// CountByStatus returns a map of status → count for expenditures of the
// given kind (`purchase` or `expense`). Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_expenditure_workspace_type_status
//	  ON expenditure(workspace_id, expenditure_type, status) WHERE active = true;
func (r *PostgresExpenditureRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(ex.status, 'unknown'), COUNT(*)::bigint
		FROM expenditure ex
		WHERE ex.active = true
		  AND ex.expenditure_type = $2
		  AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		GROUP BY ex.status`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 6)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan expenditure count row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure count rows: %w", err)
	}
	return out, nil
}

// SumOpenByStatus returns the sum (centavos) of total_amount per status for
// open (non-paid, non-cancelled) expenditures of the given kind.
// Workspace-scoped.
func (r *PostgresExpenditureRepository) SumOpenByStatus(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(ex.status, 'unknown'), COALESCE(SUM(ex.total_amount), 0)::bigint
		FROM expenditure ex
		WHERE ex.active = true
		  AND ex.expenditure_type = $2
		  AND ex.status NOT IN ('paid', 'cancelled')
		  AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		GROUP BY ex.status`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 6)
	for rows.Next() {
		var (
			status string
			sum    int64
		)
		if scanErr := rows.Scan(&status, &sum); scanErr != nil {
			return nil, fmt.Errorf("failed to scan expenditure open-sum row: %w", scanErr)
		}
		out[status] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure open-sum rows: %w", err)
	}
	return out, nil
}

// TopBySupplier returns the top suppliers by total_amount for the given kind.
// Workspace-scoped, centavos.
func (r *PostgresExpenditureRepository) TopBySupplier(
	ctx context.Context,
	workspaceID string,
	kind string,
	limit int32,
) ([]TopSupplierRow, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	// LEFT JOIN supplier so we can render a friendly name when present.
	const query = `
		SELECT
			ex.supplier_id,
			COALESCE(s.name, ex.supplier_id),
			COALESCE(SUM(ex.total_amount), 0)::bigint AS total
		FROM expenditure ex
		LEFT JOIN supplier s ON s.id = ex.supplier_id
		WHERE ex.active = true
		  AND ex.expenditure_type = $2
		  AND ex.supplier_id IS NOT NULL
		  AND ex.supplier_id <> ''
		  AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		GROUP BY ex.supplier_id, s.name
		ORDER BY total DESC NULLS LAST
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]TopSupplierRow, 0, limit)
	for rows.Next() {
		var row TopSupplierRow
		if scanErr := rows.Scan(&row.SupplierID, &row.SupplierName, &row.Total); scanErr != nil {
			return nil, fmt.Errorf("failed to scan top-supplier row: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top-supplier rows: %w", err)
	}
	return out, nil
}

// RecentByDate returns the most recent expenditures of the given kind,
// newest-first. Workspace-scoped.
func (r *PostgresExpenditureRepository) RecentByDate(
	ctx context.Context,
	workspaceID string,
	kind string,
	limit int32,
) ([]*expenditurepb.Expenditure, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT to_jsonb(ex) AS row
		FROM expenditure ex
		WHERE ex.active = true
		  AND ex.expenditure_type = $2
		  AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		ORDER BY COALESCE(ex.expenditure_date, ex.date_created) DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent expenditures: %w", err)
	}
	defer rows.Close()

	out := make([]*expenditurepb.Expenditure, 0, limit)
	for rows.Next() {
		var rowJSON []byte
		if scanErr := rows.Scan(&rowJSON); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent expenditure row: %w", scanErr)
		}
		var rowMap map[string]any
		if err := json.Unmarshal(rowJSON, &rowMap); err != nil {
			log.Printf("WARN: unmarshal recent expenditure row: %v", err)
			continue
		}
		// timestamp columns might come back as RFC3339 strings; that's fine for
		// protojson.Unmarshal which accepts both timestamp string and millis.
		clean, err := json.Marshal(rowMap)
		if err != nil {
			log.Printf("WARN: re-marshal recent expenditure row: %v", err)
			continue
		}
		ex := &expenditurepb.Expenditure{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(clean, ex); err != nil {
			log.Printf("WARN: protojson unmarshal recent expenditure: %v", err)
			continue
		}
		out = append(out, ex)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent expenditure rows: %w", err)
	}
	return out, nil
}

// SumByMonth returns one TimeBucket per calendar month in [from, to]
// (snapped to month start), with each value being the sum (centavos) of
// expenditures whose expenditure_date falls in that month.
// Workspace-scoped, kind-filtered.
func (r *PostgresExpenditureRepository) SumByMonth(
	ctx context.Context,
	workspaceID string,
	kind string,
	from, to time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if from.After(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	const query = `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', $3::timestamp),
				date_trunc('month', $4::timestamp),
				interval '1 month'
			) AS bucket
		)
		SELECT m.bucket,
		       COALESCE(SUM(ex.total_amount), 0)::bigint
		FROM months m
		LEFT JOIN expenditure ex
		  ON ex.active = true
		 AND ex.expenditure_type = $2
		 AND ex.expenditure_date >= m.bucket
		 AND ex.expenditure_date <  m.bucket + interval '1 month'
		 AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		GROUP BY m.bucket
		ORDER BY m.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, from, to)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]TimeBucket, 0, 12)
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan expenditure-by-month row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure-by-month rows: %w", err)
	}
	return out, nil
}

// SumByCategory groups completed/approved expenditures by category_id within
// a date window and returns category_id → centavo total. Used for the expense
// "spend by category" widget. Workspace-scoped.
func (r *PostgresExpenditureRepository) SumByCategory(
	ctx context.Context,
	workspaceID string,
	kind string,
	from, to time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT
			COALESCE(NULLIF(ex.expenditure_category_id, ''), 'uncategorized'),
			COALESCE(SUM(ex.total_amount), 0)::bigint
		FROM expenditure ex
		WHERE ex.active = true
		  AND ex.expenditure_type = $2
		  AND ex.status NOT IN ('cancelled')
		  AND ex.expenditure_date >= $3
		  AND ex.expenditure_date <  $4
		  AND ($1::text IS NULL OR $1::text = '' OR ex.workspace_id = $1)
		GROUP BY 1`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, from, to)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 8)
	for rows.Next() {
		var (
			cat string
			sum int64
		)
		if scanErr := rows.Scan(&cat, &sum); scanErr != nil {
			return nil, fmt.Errorf("failed to scan expenditure-by-category row: %w", scanErr)
		}
		out[cat] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure-by-category rows: %w", err)
	}
	return out, nil
}
