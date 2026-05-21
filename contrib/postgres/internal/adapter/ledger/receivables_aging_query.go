//go:build postgresql

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
// All user-provided values use parameterized queries ($1, $2, ...).
//
// The query calculates outstanding balances for each revenue record as of a
// point-in-time date ($1), using payment_date < $1+1day (not status) for accuracy.
// Results are grouped by a single row dimension and bucketed into 5 aging bands.
func buildReceivablesAgingQuery(tc TableConfig, req *agingpb.ReceivablesAgingRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimension.
	rowDim := normalizeAgingDimension(req.GetRowDimension())
	if !validAgingDimensions[rowDim] {
		rowDim = "client"
	}

	dimConfig := getAgingDimensionConfig(tc, rowDim)

	// Default as_of_date to today if not provided.
	asOfDate := req.GetAsOfDate()
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	// Build parameter list.
	// $1 = as_of_date (text, YYYY-MM-DD)
	// $2 = client_id (text or NULL)
	// $3 = location_id (text or NULL)
	// $4 = revenue_category_id (text or NULL)
	// $5 = currency (text or NULL)
	// $6 = start_date (text or NULL) — filter revenues created after
	// $7 = end_date (text or NULL) — filter revenues created before
	// $8 = workspace_id (text or NULL)
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
            ELSE (($1::date) - TO_TIMESTAMP(r.due_date / 1000.0)::date)
        END AS days_overdue
    FROM %s r
    LEFT JOIN %s tc
        ON tc.revenue_id = r.id
        AND tc.payment_date < ($1::date + interval '1 day')
    %s
    WHERE r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR r.client_id = $2)
        AND ($3::text IS NULL OR r.location_id = $3)
        AND ($4::text IS NULL OR r.revenue_category_id = $4)
        AND ($5::text IS NULL OR r.currency = $5)
        AND ($6::text IS NULL OR r.revenue_date >= $6::date)
        AND ($7::text IS NULL OR r.revenue_date < ($7::date + interval '1 day'))
        AND ($8::text IS NULL OR r.workspace_id = $8)
    GROUP BY r.id, r.total_amount, r.due_date, %s
    HAVING r.total_amount - COALESCE(SUM(tc.amount), 0) > 0
)
SELECT
    row_key,
    row_id,
    SUM(CASE WHEN days_overdue <= 0 THEN balance ELSE 0 END)::bigint AS current_amount,
    SUM(CASE WHEN days_overdue BETWEEN 1 AND 30 THEN balance ELSE 0 END)::bigint AS days_1_30,
    SUM(CASE WHEN days_overdue BETWEEN 31 AND 60 THEN balance ELSE 0 END)::bigint AS days_31_60,
    SUM(CASE WHEN days_overdue BETWEEN 61 AND 90 THEN balance ELSE 0 END)::bigint AS days_61_90,
    SUM(CASE WHEN days_overdue > 90 THEN balance ELSE 0 END)::bigint AS days_over_90,
    SUM(balance)::bigint AS total_outstanding,
    COUNT(*)::int AS invoice_count
FROM outstanding
GROUP BY row_key, row_id
ORDER BY row_key`,
		dimConfig.selectKey, dimConfig.selectID,
		tc.Revenue,
		tc.TreasuryCollection,
		dimConfig.extraJoins,
		dimConfig.groupBy,
	)

	return query, args
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

		// Accumulate summary buckets.
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
