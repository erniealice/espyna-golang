//go:build sqlserver

package ledger

import (
	"database/sql"
	"fmt"
	"time"

	payagingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"
)

// payablesAgingDimensionConfig defines SQL fragments for one row axis of the payables aging report.
type payablesAgingDimensionConfig struct {
	selectKey  string
	selectID   string
	groupBy    string
	extraJoins string
}

// validPayablesAgingDimensions is a whitelist of allowed dimension values to prevent SQL injection.
var validPayablesAgingDimensions = map[string]bool{
	"supplier":             true,
	"supplier_category":    true,
	"supplierCategory":     true,
	"location":             true,
	"location_area":        true,
	"locationArea":         true,
	"expenditure_category": true,
	"expenditureCategory":  true,
}

// normalizePayablesAgingDimension converts camelCase dimension keys to snake_case.
func normalizePayablesAgingDimension(dim string) string {
	switch dim {
	case "supplierCategory":
		return "supplier_category"
	case "locationArea":
		return "location_area"
	case "expenditureCategory":
		return "expenditure_category"
	default:
		return dim
	}
}

// getPayablesAgingDimensionConfig returns SQL fragments for the requested payables aging dimension.
func getPayablesAgingDimensionConfig(tc TableConfig, dimension string) payablesAgingDimensionConfig {
	switch dimension {
	case "supplier":
		return payablesAgingDimensionConfig{
			selectKey:  "COALESCE(s.name, 'Unassigned') AS row_key",
			selectID:   "COALESCE(e.vendor_id, '__none__') AS row_id",
			groupBy:    "e.vendor_id, s.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s s ON s.id = e.vendor_id", tc.Supplier),
		}
	case "supplier_category":
		return payablesAgingDimensionConfig{
			selectKey: "COALESCE(cat.name, 'Unassigned') AS row_key",
			selectID:  "COALESCE(sc.category_id, '__none__') AS row_id",
			groupBy:   "sc.category_id, cat.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s s ON s.id = e.vendor_id LEFT JOIN %s sc ON sc.id = s.category_id LEFT JOIN %s cat ON cat.id = sc.category_id",
				tc.Supplier, tc.SupplierCategory, tc.Category),
		}
	case "location":
		return payablesAgingDimensionConfig{
			selectKey:  "COALESCE(l.name, 'Unassigned') AS row_key",
			selectID:   "COALESCE(e.location_id, '__none__') AS row_id",
			groupBy:    "e.location_id, l.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = e.location_id", tc.Location),
		}
	case "location_area":
		return payablesAgingDimensionConfig{
			selectKey: "COALESCE(la.name, 'Unassigned') AS row_key",
			selectID:  "COALESCE(l.location_area_id, '__none__') AS row_id",
			groupBy:   "l.location_area_id, la.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s l ON l.id = e.location_id LEFT JOIN %s la ON la.id = l.location_area_id",
				tc.Location, tc.LocationArea),
		}
	case "expenditure_category":
		return payablesAgingDimensionConfig{
			selectKey:  "COALESCE(ec.name, 'Unassigned') AS row_key",
			selectID:   "COALESCE(e.expenditure_category_id, '__none__') AS row_id",
			groupBy:    "e.expenditure_category_id, ec.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s ec ON ec.id = e.expenditure_category_id", tc.ExpenditureCategory),
		}
	default:
		return getPayablesAgingDimensionConfig(tc, "supplier")
	}
}

// buildPayablesAgingQuery constructs the payables aging SQL query and its parameter args.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - CAST(@p1 AS date) replaces $1::date.
//   - DATEADD(day, 1, CAST(@p1 AS date)) replaces ($1::date + interval '1 day').
//   - TO_TIMESTAMP(e.due_date / 1000.0)::date → CAST(DATEADD(s, e.due_date / 1000, '19700101') AS date).
//   - NULLIF(td.payment_date, ”)::date → NULLIF(td.payment_date, ”) with date comparison.
//   - active = true → active = 1.
//   - ::bigint, ::int → CAST(… AS bigint), CAST(… AS int).
//   - DATEDIFF(day, due, as_of) replaces (($1::date) - due_date).
func buildPayablesAgingQuery(tc TableConfig, req *payagingpb.PayablesAgingRequest, workspaceID string) (string, []any) {
	rowDim := normalizePayablesAgingDimension(req.GetRowDimension())
	if !validPayablesAgingDimensions[rowDim] {
		rowDim = "supplier"
	}

	dimConfig := getPayablesAgingDimensionConfig(tc, rowDim)

	asOfDate := req.GetAsOfDate()
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	// Parameters mirror the postgres gold standard.
	// @p1 = as_of_date (text, YYYY-MM-DD)
	// @p2 = supplier_id (text or NULL)
	// @p3 = location_id (text or NULL)
	// @p4 = expenditure_category_id (text or NULL)
	// @p5 = currency (text or NULL)
	// @p6 = start_date (text or NULL)
	// @p7 = end_date (text or NULL)
	// @p8 = workspace_id (text or NULL)
	args := []any{
		asOfDate,
		nilIfEmpty(req.GetSupplierId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetExpenditureCategoryId()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(workspaceID),
	}

	// due_date is Unix millis; payment_date in treasury_disbursement is TEXT 'YYYY-MM-DD'.
	query := fmt.Sprintf(`
WITH outstanding AS (
    SELECT
        e.id AS expenditure_id,
        %s,
        %s,
        e.total_amount,
        COALESCE(SUM(td.amount), 0) AS total_paid,
        e.total_amount - COALESCE(SUM(td.amount), 0) AS balance,
        CASE
            WHEN e.due_date IS NULL THEN 0
            ELSE DATEDIFF(day, CAST(DATEADD(s, e.due_date / 1000, '19700101') AS date), CAST(@p1 AS date))
        END AS days_overdue
    FROM %s e
    LEFT JOIN %s td
        ON td.expenditure_id = e.id
        AND NULLIF(td.payment_date, '') < CONVERT(varchar, DATEADD(day, 1, CAST(@p1 AS date)), 23)
    %s
    WHERE e.status != 'cancelled'
        AND e.active = 1
        AND (@p2 IS NULL OR e.vendor_id = @p2)
        AND (@p3 IS NULL OR e.location_id = @p3)
        AND (@p4 IS NULL OR e.expenditure_category_id = @p4)
        AND (@p5 IS NULL OR e.currency = @p5)
        AND (@p6 IS NULL OR e.expenditure_date >= CAST(@p6 AS date))
        AND (@p7 IS NULL OR e.expenditure_date < DATEADD(day, 1, CAST(@p7 AS date)))
        AND (@p8 IS NULL OR e.workspace_id = @p8)
    GROUP BY e.id, e.total_amount, e.due_date, %s
    HAVING e.total_amount - COALESCE(SUM(td.amount), 0) > 0
)
SELECT
    row_key,
    row_id,
    CAST(SUM(CASE WHEN days_overdue <= 0 THEN balance ELSE 0 END) AS bigint) AS current_amount,
    CAST(SUM(CASE WHEN days_overdue BETWEEN 1 AND 30 THEN balance ELSE 0 END) AS bigint) AS days_1_30,
    CAST(SUM(CASE WHEN days_overdue BETWEEN 31 AND 60 THEN balance ELSE 0 END) AS bigint) AS days_31_60,
    CAST(SUM(CASE WHEN days_overdue BETWEEN 61 AND 90 THEN balance ELSE 0 END) AS bigint) AS days_61_90,
    CAST(SUM(CASE WHEN days_overdue > 90 THEN balance ELSE 0 END) AS bigint) AS days_over_90,
    CAST(SUM(balance) AS bigint) AS total_outstanding,
    CAST(COUNT(*) AS int) AS invoice_count
FROM outstanding
GROUP BY row_key, row_id
ORDER BY row_key`,
		dimConfig.selectKey, dimConfig.selectID,
		tc.Expenditure,
		tc.TreasuryDisbursement,
		dimConfig.extraJoins,
		dimConfig.groupBy,
	)

	return query, args
}

// scanPayablesAgingRows scans SQL result rows into proto aging rows and computes the summary buckets.
func scanPayablesAgingRows(rows *sql.Rows) ([]*payagingpb.PayablesAgingRow, *payagingpb.PayablesAgingBuckets, error) {
	var pbRows []*payagingpb.PayablesAgingRow
	summaryBuckets := &payagingpb.PayablesAgingBuckets{}

	for rows.Next() {
		var (
			rowKey           string
			rowID            sql.NullString
			currentAmt       int64
			days1_30         int64
			days31_60        int64
			days61_90        int64
			daysOver90       int64
			totalOutstanding int64
			invoiceCount     int32
		)
		if err := rows.Scan(
			&rowKey,
			&rowID,
			&currentAmt,
			&days1_30,
			&days31_60,
			&days61_90,
			&daysOver90,
			&totalOutstanding,
			&invoiceCount,
		); err != nil {
			return nil, nil, err
		}

		row := &payagingpb.PayablesAgingRow{
			RowKey: rowKey,
			Buckets: &payagingpb.PayablesAgingBuckets{
				Current:     currentAmt,
				Days_1_30:   days1_30,
				Days_31_60:  days31_60,
				Days_61_90:  days61_90,
				DaysOver_90: daysOver90,
			},
			TotalOutstanding: totalOutstanding,
			InvoiceCount:     invoiceCount,
		}
		if rowID.Valid {
			row.RowId = &rowID.String
		}
		pbRows = append(pbRows, row)

		summaryBuckets.Current += currentAmt
		summaryBuckets.Days_1_30 += days1_30
		summaryBuckets.Days_31_60 += days31_60
		summaryBuckets.Days_61_90 += days61_90
		summaryBuckets.DaysOver_90 += daysOver90
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return pbRows, summaryBuckets, nil
}

// buildPayablesAgingSummary computes report-level totals from assembled rows.
func buildPayablesAgingSummary(agingRows []*payagingpb.PayablesAgingRow, summaryBuckets *payagingpb.PayablesAgingBuckets, req *payagingpb.PayablesAgingRequest) *payagingpb.PayablesAgingSummary {
	s := &payagingpb.PayablesAgingSummary{
		Buckets: summaryBuckets,
	}
	for _, row := range agingRows {
		s.GrandTotalOutstanding += row.TotalOutstanding
		s.TotalInvoiceCount += row.InvoiceCount
	}
	if req != nil {
		if req.AsOfDate != nil {
			d := req.GetAsOfDate()
			s.AsOfDate = &d
		}
		if req.StartDate != nil {
			sd := req.GetStartDate()
			s.StartDate = &sd
		}
		if req.EndDate != nil {
			ed := req.GetEndDate()
			s.EndDate = &ed
		}
		if req.Currency != nil {
			s.Currency = req.GetCurrency()
		}
	}
	return s
}
