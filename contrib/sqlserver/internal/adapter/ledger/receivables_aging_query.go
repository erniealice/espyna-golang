//go:build sqlserver

package ledger

import (
	"database/sql"
	"fmt"
	"time"

	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
)

// agingDimensionConfig defines SQL fragments for one row axis of the aging report.
type agingDimensionConfig struct {
	selectKey  string
	selectID   string
	groupBy    string
	extraJoins string
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

// normalizeAgingDimension converts camelCase dimension keys to snake_case.
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
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - ($1::date) → CAST(@p1 AS date).
//   - TO_TIMESTAMP(r.due_date / 1000.0)::date → CAST(DATEADD(ms, r.due_date % 1000, DATEADD(s, r.due_date / 1000, '19700101')) AS date).
//   - interval '1 day' → +1 in DATEADD or comparison.
//   - active = true → active = 1.
//   - SUM(x) FILTER (WHERE c) → SUM(CASE WHEN c THEN x END) (already CASE in aging sums).
//   - ::bigint, ::int → CAST(… AS bigint), CAST(… AS int).
//   - preserve *100 math — amounts are already centavos here, no centavo conversion needed.
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

	// Parameters mirror the postgres gold standard.
	// @p1 = as_of_date (text, YYYY-MM-DD)
	// @p2 = client_id (text or NULL)
	// @p3 = location_id (text or NULL)
	// @p4 = revenue_category_id (text or NULL)
	// @p5 = currency (text or NULL)
	// @p6 = start_date (text or NULL)
	// @p7 = end_date (text or NULL)
	// @p8 = workspace_id (text or NULL)
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

	// due_date is stored as Unix milliseconds (int64). Convert to date:
	//   CAST(DATEADD(s, r.due_date / 1000, '19700101') AS date)
	// payment_date < as_of_date + 1 day: payment_date < DATEADD(day, 1, CAST(@p1 AS date))
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
            ELSE DATEDIFF(day, CAST(DATEADD(s, r.due_date / 1000, '19700101') AS date), CAST(@p1 AS date))
        END AS days_overdue
    FROM %s r
    LEFT JOIN %s tc
        ON tc.revenue_id = r.id
        AND tc.payment_date < DATEADD(day, 1, CAST(@p1 AS date))
    %s
    WHERE r.status != 'cancelled'
        AND r.active = 1
        AND (@p2 IS NULL OR r.client_id = @p2)
        AND (@p3 IS NULL OR r.location_id = @p3)
        AND (@p4 IS NULL OR r.revenue_category_id = @p4)
        AND (@p5 IS NULL OR r.currency = @p5)
        AND (@p6 IS NULL OR r.revenue_date >= CAST(@p6 AS date))
        AND (@p7 IS NULL OR r.revenue_date < DATEADD(day, 1, CAST(@p7 AS date)))
        AND (@p8 IS NULL OR r.workspace_id = @p8)
    GROUP BY r.id, r.total_amount, r.due_date, %s
    HAVING r.total_amount - COALESCE(SUM(tc.amount), 0) > 0
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
