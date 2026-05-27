//go:build sqlserver

package expenditure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: the SQL Server adapter
// MUST return EXACTLY the named row types declared by the service-layer dashboard
// package. Aliasing here keeps signatures identical so the compile-time assertion
// in expenditure_dashboard_assertions.go succeeds.
type (
	TimeBucket     = expendituredash.TimeBucket
	TopSupplierRow = expendituredash.TopSupplierRow
)

// expenditureStatusAggregate holds the per-status metrics derived in a single
// pass over the kind-filtered, workspace-scoped expenditure rows.
type expenditureStatusAggregate struct {
	CountByStatus   map[string]int64
	OpenSumByStatus map[string]int64
}

// expenditureStatusAggregateQuery is the consolidated grouped CTE backing both
// CountByStatus and SumOpenByStatus.
//
// SQL Server translation from postgres gold standard:
//   - $N → @pN
//   - active = true → active = 1
//   - $1::text IS NULL OR $1::text = ” → @p1 IS NULL OR @p1 = ”
//   - FILTER (WHERE is_open) → SUM(CASE WHEN is_open = 1 THEN total_amount END)
//   - ::bigint cast → no cast needed (SQL Server infers BIGINT from SUM)
//   - boolean (is_open) AS BIT: CASE WHEN ... THEN 1 ELSE 0 END
const expenditureStatusAggregateQuery = `
	WITH base AS (
		SELECT
			COALESCE(ex.status, 'unknown') AS status,
			ex.total_amount,
			CASE WHEN ex.status NOT IN ('paid', 'cancelled') THEN 1 ELSE 0 END AS is_open
		FROM [expenditure] ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = @p2
		  AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)
	)
	SELECT
		status,
		COUNT(*) AS cnt,
		COALESCE(SUM(CASE WHEN is_open = 1 THEN total_amount END), 0) AS open_sum
	FROM base
	GROUP BY status`

// runExpenditureStatusAggregate executes the consolidated per-status CTE once
// and returns both metric maps. Workspace-scoped, kind-filtered.
func (r *SQLServerExpenditureRepository) runExpenditureStatusAggregate(
	ctx context.Context,
	workspaceID string,
	kind string,
) (expenditureStatusAggregate, error) {
	if r.db == nil {
		return expenditureStatusAggregate{}, fmt.Errorf("database connection is not available")
	}

	rows, err := r.db.QueryContext(ctx, expenditureStatusAggregateQuery, workspaceID, kind)
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
func (r *SQLServerExpenditureRepository) CountByStatus(
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
// open (non-paid, non-cancelled) expenditures of the given kind.
func (r *SQLServerExpenditureRepository) SumOpenByStatus(
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
// Workspace-scoped, centavos.
//
// SQL Server translation:
//   - LIMIT $3 → TOP (@p3) in the SELECT clause
//   - NULLS LAST → not needed (SQL Server NULL sorts last for DESC by default)
//   - $N → @pN
func (r *SQLServerExpenditureRepository) TopBySupplier(
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
		SELECT TOP (@p3)
			ex.supplier_id,
			COALESCE(s.[name], ex.supplier_id),
			COALESCE(SUM(ex.total_amount), 0) AS total
		FROM [expenditure] ex
		LEFT JOIN [supplier] s ON s.id = ex.supplier_id
		WHERE ex.active = 1
		  AND ex.expenditure_type = @p2
		  AND ex.supplier_id IS NOT NULL
		  AND ex.supplier_id <> ''
		  AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)
		GROUP BY ex.supplier_id, s.[name]
		ORDER BY total DESC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
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

// RecentByDate returns the most recent expenditures of the given kind,
// newest-first. Workspace-scoped.
//
// SQL Server translation:
//   - to_jsonb(ex) → SELECT columns individually (no JSON shorthand in SQL Server)
//   - LIMIT $3 → TOP (@p3)
//   - $N → @pN
func (r *SQLServerExpenditureRepository) RecentByDate(
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

	// Select all columns individually; alias matches proto field names via
	// DenormalizeKeys / protojson round-trip. SQL Server 2016+ supports
	// FOR JSON PATH but the column-by-column approach avoids driver/version
	// assumptions and mirrors the existing DenormalizeKeys pattern used
	// everywhere else in this adapter package.
	const query = `
		SELECT TOP (@p3)
			ex.id,
			ex.name,
			ex.expenditure_type,
			ex.supplier_id,
			ex.expenditure_date,
			ex.expenditure_date_string,
			ex.total_amount,
			ex.currency,
			ex.status,
			ex.reference_number,
			ex.notes,
			ex.expenditure_category_id,
			ex.location_id,
			ex.payment_terms,
			ex.due_date,
			ex.approved_by,
			ex.purchase_order_id,
			ex.run_id,
			ex.workspace_id,
			ex.active,
			ex.date_created,
			ex.date_modified
		FROM [expenditure] ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = @p2
		  AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)
		ORDER BY COALESCE(ex.expenditure_date, ex.date_created) DESC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent expenditures: %w", err)
	}
	defer rows.Close()

	out := make([]*expenditurepb.Expenditure, 0, limit)
	for rows.Next() {
		var (
			id                    string
			name                  *string
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
			workspaceIDVal        *string
			active                bool
			dateCreated           time.Time
			dateModified          time.Time
		)
		if scanErr := rows.Scan(
			&id, &name, &expenditureType, &supplierID,
			&expenditureDate, &expenditureDateString,
			&totalAmount, &currency, &status, &referenceNumber,
			&notes, &expenditureCategoryID, &locationID, &paymentTerms,
			&dueDate, &approvedBy, &purchaseOrderID, &runID,
			&workspaceIDVal, &active, &dateCreated, &dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent expenditure row: %w", scanErr)
		}

		rowMap := map[string]any{
			"id":                      id,
			"name":                    name,
			"expenditure_type":        expenditureType,
			"supplier_id":             supplierID,
			"expenditure_date":        expenditureDate,
			"expenditure_date_string": expenditureDateString,
			"total_amount":            totalAmount,
			"currency":                currency,
			"status":                  status,
			"reference_number":        referenceNumber,
			"notes":                   notes,
			"expenditure_category_id": expenditureCategoryID,
			"location_id":             locationID,
			"payment_terms":           paymentTerms,
			"due_date":                dueDate,
			"approved_by":             approvedBy,
			"purchase_order_id":       purchaseOrderID,
			"run_id":                  runID,
			"workspace_id":            workspaceIDVal,
			"active":                  active,
			"date_created":            dateCreated,
			"date_modified":           dateModified,
		}

		clean, err := json.Marshal(sqlserverCore.DenormalizeKeys(rowMap))
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
//
// SQL Server translation:
//   - generate_series(...) → recursive CTE (anchor = month-snapped from,
//     recursion adds 1 month each step, terminates when bucket > to)
//   - date_trunc('month', $3::timestamp) → DATEFROMPARTS(YEAR(@p3),MONTH(@p3),1)
//   - interval '1 month' → DATEADD(month, 1, bucket)
//   - $N → @pN
//   - LEFT JOIN expenditure ... AND active = true → active = 1
func (r *SQLServerExpenditureRepository) SumByMonth(
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
			SELECT DATEFROMPARTS(YEAR(@p3), MONTH(@p3), 1) AS bucket
			UNION ALL
			SELECT DATEADD(month, 1, bucket)
			FROM months
			WHERE DATEADD(month, 1, bucket) <= DATEFROMPARTS(YEAR(@p4), MONTH(@p4), 1)
		)
		SELECT m.bucket,
		       COALESCE(SUM(ex.total_amount), 0) AS total
		FROM months m
		LEFT JOIN [expenditure] ex
		  ON ex.active = 1
		 AND ex.expenditure_type = @p2
		 AND ex.expenditure_date >= m.bucket
		 AND ex.expenditure_date < DATEADD(month, 1, m.bucket)
		 AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)
		GROUP BY m.bucket
		ORDER BY m.bucket ASC
		OPTION (MAXRECURSION 120)`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenditure-by-month: %w", err)
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

// SumByCategory groups approved/non-cancelled expenditures by category_id within
// a date window and returns category_id → centavo total. Workspace-scoped.
//
// SQL Server translation:
//   - NULLIF(ex.expenditure_category_id, ”) → NULLIF(ex.expenditure_category_id, ”)
//     (same function supported in SQL Server)
//   - $N → @pN
//   - active = true → active = 1
//   - GROUP BY 1 → GROUP BY COALESCE(NULLIF(...), 'uncategorized')
func (r *SQLServerExpenditureRepository) SumByCategory(
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
		FROM [expenditure] ex
		WHERE ex.active = 1
		  AND ex.expenditure_type = @p2
		  AND ex.status NOT IN ('cancelled')
		  AND ex.expenditure_date >= @p3
		  AND ex.expenditure_date <  @p4
		  AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)
		GROUP BY COALESCE(NULLIF(ex.expenditure_category_id, ''), 'uncategorized')`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, kind, from, to)
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
