//go:build mysql

package ledger

import (
	"database/sql"
	"fmt"
	"time"

	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
)

// agingDimensionConfig defines SQL fragments for one row axis of the aging report.
type agingDimensionConfig struct {
	selectKey  string // SQL expression yielding a human-readable label
	selectID   string // SQL expression yielding an entity ID
	groupBy    string // GROUP BY fragment (may be multiple columns)
	extraJoins string // Additional JOINs required by this dimension
}

// validAgingDimensions is a whitelist of allowed dimension values to prevent SQL injection.
var validAgingDimensions = map[string]bool{
	"client":          true,
	"client_category": true,
	"clientCategory":  true,
	"location":        true,
	"location_area":   true,
	"locationArea":    true,
}

// normalizeAgingDimension converts camelCase dimension keys to snake_case for SQL switch matching.
func normalizeAgingDimension(dim string) string {
	switch dim {
	case "clientCategory":
		return "client_category"
	case "locationArea":
		return "location_area"
	default:
		return dim
	}
}

// getAgingDimensionConfig returns SQL fragments for the requested aging dimension.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
func getAgingDimensionConfig(tc TableConfig, dimension string) agingDimensionConfig {
	switch dimension {
	case "client":
		return agingDimensionConfig{
			selectKey:  "COALESCE(cl.name, 'Unassigned') AS row_key",
			selectID:   "COALESCE(r.client_id, '__none__') AS row_id",
			groupBy:    "r.client_id, cl.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s cl ON cl.id = r.client_id", tc.Client),
		}
	case "client_category":
		return agingDimensionConfig{
			selectKey: "COALESCE(cat.name, 'Unassigned') AS row_key",
			selectID:  "COALESCE(cc.category_id, '__none__') AS row_id",
			groupBy:   "cc.category_id, cat.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s cl ON cl.id = r.client_id LEFT JOIN %s cc ON cc.id = cl.category_id LEFT JOIN %s cat ON cat.id = cc.category_id",
				tc.Client, tc.ClientCategory, tc.Category),
		}
	case "location":
		return agingDimensionConfig{
			selectKey:  "COALESCE(l.name, 'Unassigned') AS row_key",
			selectID:   "COALESCE(r.location_id, '__none__') AS row_id",
			groupBy:    "r.location_id, l.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = r.location_id", tc.Location),
		}
	case "location_area":
		return agingDimensionConfig{
			selectKey: "COALESCE(la.name, 'Unassigned') AS row_key",
			selectID:  "COALESCE(l.location_area_id, '__none__') AS row_id",
			groupBy:   "l.location_area_id, la.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s l ON l.id = r.location_id LEFT JOIN %s la ON la.id = l.location_area_id",
				tc.Location, tc.LocationArea),
		}
	default:
		return getAgingDimensionConfig(tc, "client")
	}
}

// buildReceivablesAgingQuery constructs the aging SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries (?).
//
// Dialect differences from postgres gold standard:
//   - $N → ? (positional, same left-to-right order)
//   - $1::date → DATE(?): MySQL casts via DATE()
//   - TO_TIMESTAMP(x / 1000.0) → FROM_UNIXTIME(x / 1000) (millis → epoch)
//   - interval '1 day' → INTERVAL 1 DAY
//   - ::bigint / ::int casts removed (MySQL uses SIGNED/UNSIGNED or infers)
//   - SUM(x) FILTER (WHERE c) → SUM(CASE WHEN c THEN x END) (MySQL has no FILTER)
//   - $N::text IS NULL → ? IS NULL (MySQL driver passes NULL directly)
func buildReceivablesAgingQuery(tc TableConfig, req *agingpb.ReceivablesAgingRequest, workspaceID string) (string, []any) {
	rowDim := normalizeAgingDimension(req.GetRowDimension())
	if !validAgingDimensions[rowDim] {
		rowDim = "client"
	}

	dimConfig := getAgingDimensionConfig(tc, rowDim)

	asOfDate := req.GetAsOfDate()
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	// Arg order preserved from postgres gold standard:
	// ? = as_of_date (text, YYYY-MM-DD)
	// ? = client_id (text or NULL)
	// ? = location_id (text or NULL)
	// ? = revenue_category_id (text or NULL)
	// ? = currency (text or NULL)
	// ? = start_date (text or NULL)
	// ? = end_date (text or NULL)
	// ? = workspace_id (text or NULL)
	args := []any{
		asOfDate,
		nilIfEmpty(req.GetClientId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetRevenueCategoryId()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(workspaceID),
	}

	// MySQL translations:
	//   $1::date → DATE(?)
	//   TO_TIMESTAMP(r.due_date / 1000.0)::date → DATE(FROM_UNIXTIME(r.due_date / 1000))
	//   $1::date + interval '1 day' → DATE_ADD(DATE(?), INTERVAL 1 DAY)
	//   $N::text IS NULL → ? IS NULL
	//   SUM(...)::bigint → SUM(...) (MySQL uses SIGNED arithmetic)
	//   COUNT(*)::int → COUNT(*)
	query := fmt.Sprintf(`
WITH outstanding AS (
    SELECT
        r.id AS revenue_id,
        %s,
        %s,
        r.total_amount,
        COALESCE(SUM(tc.amount), 0) AS total_paid,
        r.total_amount - COALESCE(SUM(tc.amount), 0) AS balance,
        CASE
            WHEN r.due_date IS NULL THEN 0
            ELSE DATEDIFF(DATE(?), DATE(FROM_UNIXTIME(r.due_date / 1000)))
        END AS days_overdue
    FROM %s r
    LEFT JOIN %s tc
        ON tc.revenue_id = r.id
        AND tc.payment_date < DATE_ADD(DATE(?), INTERVAL 1 DAY)
    %s
    WHERE r.status != 'cancelled'
        AND r.active = 1
        AND (? IS NULL OR r.client_id = ?)
        AND (? IS NULL OR r.location_id = ?)
        AND (? IS NULL OR r.revenue_category_id = ?)
        AND (? IS NULL OR r.currency = ?)
        AND (? IS NULL OR r.revenue_date >= ?)
        AND (? IS NULL OR r.revenue_date < DATE_ADD(DATE(?), INTERVAL 1 DAY))
        AND (? IS NULL OR r.workspace_id = ?)
    GROUP BY r.id, r.total_amount, r.due_date, %s
    HAVING r.total_amount - COALESCE(SUM(tc.amount), 0) > 0
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
		tc.Revenue,
		tc.TreasuryCollection,
		dimConfig.extraJoins,
		dimConfig.groupBy,
	)

	// MySQL requires each ? to be bound once. Expand the repeated params that
	// appear twice in the WHERE clause (as_of_date is used twice in DATEDIFF and
	// in the payment_date filter; each optional param uses (? IS NULL OR col = ?)).
	expandedArgs := []any{
		args[0],          // DATEDIFF DATE(?)
		args[0],          // DATE_ADD DATE(?)
		args[1], args[1], // client_id IS NULL / = ?
		args[2], args[2], // location_id IS NULL / = ?
		args[3], args[3], // revenue_category_id IS NULL / = ?
		args[4], args[4], // currency IS NULL / = ?
		args[5], args[5], // start_date IS NULL / >= ?
		args[6], args[6], args[6], // end_date IS NULL / < DATE_ADD(DATE(?), ...)
		args[7], args[7], // workspace_id IS NULL / = ?
	}

	return query, expandedArgs
}

// scanAgingRows scans SQL result rows into proto aging rows and computes the summary buckets.
func scanAgingRows(rows *sql.Rows) ([]*agingpb.ReceivablesAgingRow, *agingpb.AgingBuckets, error) {
	var pbRows []*agingpb.ReceivablesAgingRow
	summaryBuckets := &agingpb.AgingBuckets{}

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

		row := &agingpb.ReceivablesAgingRow{
			RowKey: rowKey,
			Buckets: &agingpb.AgingBuckets{
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

// buildAgingSummary computes report-level totals from assembled rows.
func buildAgingSummary(agingRows []*agingpb.ReceivablesAgingRow, summaryBuckets *agingpb.AgingBuckets, req *agingpb.ReceivablesAgingRequest) *agingpb.ReceivablesAgingSummary {
	s := &agingpb.ReceivablesAgingSummary{
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
