//go:build mysql

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
// Dialect differences from postgres gold standard:
//   - $N → ? (positional, same left-to-right order)
//   - $1::date → DATE(?)
//   - TO_TIMESTAMP(e.due_date / 1000.0)::date → DATE(FROM_UNIXTIME(e.due_date / 1000))
//   - $1::date + interval '1 day' → DATE_ADD(DATE(?), INTERVAL 1 DAY)
//   - NULLIF(td.payment_date, ”)::date → NULLIF(td.payment_date, ”) (MySQL DATE comparison)
//   - ::bigint / ::int casts removed
//   - SUM(...) FILTER (WHERE c) → SUM(CASE WHEN c THEN ... END)
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

	// Base args in same left-to-right order as postgres gold standard.
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
            ELSE DATEDIFF(DATE(?), DATE(FROM_UNIXTIME(e.due_date / 1000)))
        END AS days_overdue
    FROM %s e
    LEFT JOIN %s td
        ON td.expenditure_id = e.id
        AND NULLIF(td.payment_date, '') < DATE_ADD(DATE(?), INTERVAL 1 DAY)
    %s
    WHERE e.status != 'cancelled'
        AND e.active = 1
        AND (? IS NULL OR e.vendor_id = ?)
        AND (? IS NULL OR e.location_id = ?)
        AND (? IS NULL OR e.expenditure_category_id = ?)
        AND (? IS NULL OR e.currency = ?)
        AND (? IS NULL OR e.expenditure_date >= ?)
        AND (? IS NULL OR e.expenditure_date < DATE_ADD(DATE(?), INTERVAL 1 DAY))
        AND (? IS NULL OR e.workspace_id = ?)
    GROUP BY e.id, e.total_amount, e.due_date, %s
    HAVING e.total_amount - COALESCE(SUM(td.amount), 0) > 0
)
SELECT
    row_key,
    row_id,
    SUM(CASE WHEN days_overdue <= 0 THEN balance ELSE 0 END) AS current_amount,
    SUM(CASE WHEN days_overdue BETWEEN 1 AND 30 THEN balance ELSE 0 END) AS days_1_30,
    SUM(CASE WHEN days_overdue BETWEEN 31 AND 60 THEN balance ELSE 0 END) AS days_31_60,
    SUM(CASE WHEN days_overdue BETWEEN 61 AND 90 THEN balance ELSE 0 END) AS days_61_90,
    SUM(CASE WHEN days_overdue > 90 THEN balance ELSE 0 END) AS days_over_90,
    SUM(balance) AS total_outstanding,
    COUNT(*) AS invoice_count
FROM outstanding
GROUP BY row_key, row_id
ORDER BY row_key`,
		dimConfig.selectKey, dimConfig.selectID,
		tc.Expenditure,
		tc.TreasuryDisbursement,
		dimConfig.extraJoins,
		dimConfig.groupBy,
	)

	// Expand repeated params (each ? must be bound once in MySQL).
	expandedArgs := []any{
		args[0],          // DATEDIFF DATE(?)
		args[0],          // DATE_ADD DATE(?)
		args[1], args[1], // supplier_id IS NULL / = ?
		args[2], args[2], // location_id IS NULL / = ?
		args[3], args[3], // expenditure_category_id IS NULL / = ?
		args[4], args[4], // currency IS NULL / = ?
		args[5], args[5], // start_date IS NULL / >= ?
		args[6], args[6], args[6], // end_date IS NULL / < DATE_ADD(DATE(?), ...)
		args[7], args[7], // workspace_id IS NULL / = ?
	}

	return query, expandedArgs
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
			&rowKey, &rowID,
			&currentAmt, &days1_30, &days31_60, &days61_90, &daysOver90,
			&totalOutstanding, &invoiceCount,
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
