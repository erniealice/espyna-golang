//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args resequenced)
//   - active = true → active = 1
//   - $N::text IS NULL OR ... → (? = ” OR ...)
//   - SUM(x) FILTER (WHERE c) → SUM(CASE WHEN c THEN x END) (A8 conditional agg)
//   - to_jsonb(ex) → explicit column SELECT (MySQL has no row_to_json/to_jsonb)
//   - generate_series → Go-side month iteration (MySQL 8.0 has no generate_series)
//   - ORDER BY total DESC NULLS LAST → ORDER BY total DESC (MySQL NULLs sort last by default on DESC)
//   - ::bigint casts removed (MySQL infers types)
//
// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: the MySQL adapter
// MUST return EXACTLY the named row types declared by the service-layer
// dashboard package.
//
// Centavos (total_amount) are never scaled in SQL.
package expenditure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"

	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract aliases.
type (
	TimeBucket     = expendituredash.TimeBucket
	TopSupplierRow = expendituredash.TopSupplierRow
)

// expenditureStatusAggregate holds the per-status metrics derived in a single
// consolidated query.
type expenditureStatusAggregate struct {
	CountByStatus   map[string]int64
	OpenSumByStatus map[string]int64
}

// expenditureStatusAggregateQuery is the ONE consolidated grouped CTE backing
// both CountByStatus and SumOpenByStatus (Q-DASHBOARD-FAILOPEN, Option A).
//
// Dialect changes from postgres gold standard:
//   - $1/$2 → ?/? (two positional args: workspaceID, kind)
//   - $N::text IS NULL OR ... → (? = ” OR ...)
//   - active = true → active = 1
//   - SUM(total_amount) FILTER (WHERE is_open) →
//     SUM(CASE WHEN is_open THEN total_amount END)  ← A8 conditional agg
//   - COUNT(*)::bigint → COUNT(*) (MySQL returns BIGINT by default)
//   - COALESCE(SUM(...), 0)::bigint → COALESCE(SUM(...), 0)
//
// is_open is computed inline inside the CASE since MySQL CTEs cannot reference
// computed boolean columns by alias in an aggregate expression.
const expenditureStatusAggregateQuery = `
	WITH base AS (
		SELECT
			COALESCE(ex.status, 'unknown') AS status,
			ex.total_amount,
			ex.status NOT IN ('paid', 'cancelled') AS is_open
		FROM expenditure ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = ?
		  AND (? = '' OR ex.workspace_id = ?)
	)
	SELECT
		status,
		COUNT(*)                                                           AS cnt,
		COALESCE(SUM(CASE WHEN is_open THEN total_amount END), 0)          AS open_sum
	FROM base
	GROUP BY status`

// runExpenditureStatusAggregate executes the consolidated per-status CTE once.
func (r *MySQLExpenditureRepository) runExpenditureStatusAggregate(
	ctx context.Context,
	workspaceID string,
	kind string,
) (expenditureStatusAggregate, error) {
	if r.db == nil {
		return expenditureStatusAggregate{}, fmt.Errorf("database connection is not available")
	}

	rows, err := r.db.QueryContext(ctx, expenditureStatusAggregateQuery, kind, workspaceID, workspaceID)
	if err != nil {
		return expenditureStatusAggregate{}, fmt.Errorf("failed to query expenditure status aggregate: %w", err)
	}
	defer rows.Close()

	agg := expenditureStatusAggregate{
		CountByStatus:   make(map[string]int64, 6),
		OpenSumByStatus: make(map[string]int64, 6),
	}
	for rows.Next() {
		var (
			status  string
			cnt     int64
			openSum int64
		)
		if scanErr := rows.Scan(&status, &cnt, &openSum); scanErr != nil {
			return expenditureStatusAggregate{}, fmt.Errorf("failed to scan expenditure status aggregate row: %w", scanErr)
		}
		agg.CountByStatus[status] = cnt
		if openSum != 0 {
			agg.OpenSumByStatus[status] = openSum
		}
	}
	if err := rows.Err(); err != nil {
		return expenditureStatusAggregate{}, fmt.Errorf("error iterating expenditure status aggregate rows: %w", err)
	}
	return agg, nil
}

// CountByStatus returns a map of status → count for expenditures of the given kind.
func (r *MySQLExpenditureRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	agg, err := r.runExpenditureStatusAggregate(ctx, workspaceID, kind)
	if err != nil {
		return nil, err
	}
	return agg.CountByStatus, nil
}

// SumOpenByStatus returns the sum (centavos) of total_amount per status for
// open expenditures of the given kind.
func (r *MySQLExpenditureRepository) SumOpenByStatus(
	ctx context.Context,
	workspaceID string,
	kind string,
) (map[string]int64, error) {
	agg, err := r.runExpenditureStatusAggregate(ctx, workspaceID, kind)
	if err != nil {
		return nil, err
	}
	return agg.OpenSumByStatus, nil
}

// TopBySupplier returns the top suppliers by total_amount for the given kind.
//
// Dialect: $1/$2/$3 → ?/?/?; active = true → active = 1;
// ORDER BY total DESC NULLS LAST → ORDER BY total DESC (MySQL default).
func (r *MySQLExpenditureRepository) TopBySupplier(
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

	const query = `
		SELECT
			ex.supplier_id,
			COALESCE(s.name, ex.supplier_id),
			COALESCE(SUM(ex.total_amount), 0) AS total
		FROM expenditure ex
		LEFT JOIN supplier s ON s.id = ex.supplier_id
		WHERE ex.active = 1
		  AND ex.expenditure_type = ?
		  AND ex.supplier_id IS NOT NULL
		  AND ex.supplier_id <> ''
		  AND (? = '' OR ex.workspace_id = ?)
		GROUP BY ex.supplier_id, s.name
		ORDER BY total DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, kind, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top suppliers: %w", err)
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

// RecentByDate returns the most recent expenditures of the given kind, newest-first.
//
// Dialect: to_jsonb(ex) → explicit column SELECT; $1/$2/$3 → ?/?/?;
// active = true → active = 1.
func (r *MySQLExpenditureRepository) RecentByDate(
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

	// MySQL has no to_jsonb; select columns explicitly and use DenormalizeKeys.
	const query = `
		SELECT
			ex.id, ex.date_created, ex.date_modified, ex.active,
			ex.name, ex.expenditure_type, ex.supplier_id,
			ex.expenditure_date, ex.expenditure_date_string,
			ex.total_amount, ex.currency, ex.status,
			ex.reference_number, ex.notes,
			ex.expenditure_category_id, ex.location_id,
			ex.payment_terms, ex.due_date, ex.approved_by,
			ex.purchase_order_id, ex.run_id, ex.workspace_id
		FROM expenditure ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = ?
		  AND (? = '' OR ex.workspace_id = ?)
		ORDER BY COALESCE(ex.expenditure_date, ex.date_created) DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, kind, workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent expenditures: %w", err)
	}
	defer rows.Close()

	out := make([]*expenditurepb.Expenditure, 0, limit)
	for rows.Next() {
		var (
			id                    string
			dateCreated           *time.Time
			dateModified          *time.Time
			active                bool
			name                  string
			expenditureType       *string
			supplierID            *string
			expenditureDate       *time.Time
			expenditureDateString *string
			totalAmount           int64
			currency              *string
			status                *string
			referenceNumber       *string
			notes                 *string
			expenditureCategoryID *string
			locationID            *string
			paymentTerms          *string
			dueDate               *string
			approvedBy            *string
			purchaseOrderID       *string
			runID                 *string
			workspaceIDCol        *string
		)
		if scanErr := rows.Scan(
			&id, &dateCreated, &dateModified, &active,
			&name, &expenditureType, &supplierID,
			&expenditureDate, &expenditureDateString,
			&totalAmount, &currency, &status,
			&referenceNumber, &notes,
			&expenditureCategoryID, &locationID,
			&paymentTerms, &dueDate, &approvedBy,
			&purchaseOrderID, &runID, &workspaceIDCol,
		); scanErr != nil {
			log.Printf("WARN: scan recent expenditure row: %v", scanErr)
			continue
		}

		raw := map[string]any{
			"id":           id,
			"active":       active,
			"name":         name,
			"total_amount": totalAmount,
		}
		if expenditureType != nil {
			raw["expenditure_type"] = *expenditureType
		}
		if supplierID != nil {
			raw["supplier_id"] = *supplierID
		}
		if expenditureDateString != nil {
			raw["expenditure_date_string"] = *expenditureDateString
		}
		if totalAmount != 0 {
			raw["total_amount"] = totalAmount
		}
		if currency != nil {
			raw["currency"] = *currency
		}
		if status != nil {
			raw["status"] = *status
		}
		if referenceNumber != nil {
			raw["reference_number"] = *referenceNumber
		}
		if notes != nil {
			raw["notes"] = *notes
		}
		if expenditureCategoryID != nil {
			raw["expenditure_category_id"] = *expenditureCategoryID
		}
		if locationID != nil {
			raw["location_id"] = *locationID
		}
		if paymentTerms != nil {
			raw["payment_terms"] = *paymentTerms
		}
		if dueDate != nil {
			raw["due_date"] = *dueDate
		}
		if approvedBy != nil {
			raw["approved_by"] = *approvedBy
		}
		if purchaseOrderID != nil {
			raw["purchase_order_id"] = *purchaseOrderID
		}
		if runID != nil {
			raw["run_id"] = *runID
		}
		if dateCreated != nil {
			raw["date_created"] = dateCreated.UnixMilli()
		}
		if dateModified != nil {
			raw["date_modified"] = dateModified.UnixMilli()
		}
		if expenditureDate != nil {
			raw["expenditure_date"] = expenditureDate.UnixMilli()
		}

		clean, err := json.Marshal(mysqlCore.DenormalizeKeys(raw))
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

// SumByMonth returns one TimeBucket per calendar month in [from, to].
//
// Dialect: MySQL 8.0 has no generate_series. We generate the month spine
// in Go and issue one SUM per month using a grouped query that covers
// [from, to] in a single pass, then fill in zeros for missing months.
// This avoids N round-trips while matching the postgres output contract.
//
// Query changes: $1/$2/$3/$4 → ?/?/?/?; active = true → active = 1.
func (r *MySQLExpenditureRepository) SumByMonth(
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

	// Snap to month start (matches postgres date_trunc('month', ...)).
	from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, to.Location())

	// Build the month spine in Go.
	var months []time.Time
	for cur := from; !cur.After(to); cur = cur.AddDate(0, 1, 0) {
		months = append(months, cur)
	}

	// Fetch sums for the whole window in a single query.
	const query = `
		SELECT
			DATE_FORMAT(ex.expenditure_date, '%Y-%m-01') AS bucket,
			COALESCE(SUM(ex.total_amount), 0)            AS total
		FROM expenditure ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = ?
		  AND ex.expenditure_date >= ?
		  AND ex.expenditure_date <  DATE_ADD(?, INTERVAL 1 MONTH)
		  AND (? = '' OR ex.workspace_id = ?)
		GROUP BY bucket
		ORDER BY bucket ASC`

	// to + 1 month so the last bucket is inclusive.
	toNext := to.AddDate(0, 1, 0)

	rows, err := r.db.QueryContext(ctx, query, kind, from, toNext, workspaceID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenditure-by-month: %w", err)
	}
	defer rows.Close()

	// Collect results into a map keyed by "YYYY-MM-01".
	sums := make(map[string]int64, len(months))
	for rows.Next() {
		var (
			bucketStr string
			value     int64
		)
		if scanErr := rows.Scan(&bucketStr, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan expenditure-by-month row: %w", scanErr)
		}
		sums[bucketStr] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure-by-month rows: %w", err)
	}

	// Fill spine (zero for months with no data).
	out := make([]TimeBucket, 0, len(months))
	for _, m := range months {
		key := m.Format("2006-01-02")
		out = append(out, TimeBucket{Period: m, Value: sums[key]})
	}
	return out, nil
}

// SumByCategory groups expenditures by category_id within a date window.
//
// Dialect: $1/$2/$3/$4 → ?/?/?/?; active = true → active = 1;
// NULLIF/COALESCE stays (MySQL supports both).
func (r *MySQLExpenditureRepository) SumByCategory(
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
			COALESCE(SUM(ex.total_amount), 0)
		FROM expenditure ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = ?
		  AND ex.status NOT IN ('cancelled')
		  AND ex.expenditure_date >= ?
		  AND ex.expenditure_date <  ?
		  AND (? = '' OR ex.workspace_id = ?)
		GROUP BY 1`

	rows, err := r.db.QueryContext(ctx, query, kind, from, to, workspaceID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenditure-by-category: %w", err)
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
