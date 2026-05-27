//go:build mysql

package ledger

import (
	"fmt"
	"sort"
	"strings"

	revreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
)

// pivotDimensionConfig defines SQL fragments for one axis of the pivot.
type pivotDimensionConfig struct {
	selectKey  string
	selectID   string
	groupBy    string
	extraJoins string
}

// validPivotDimensions is a whitelist of allowed dimension values to prevent SQL injection.
var validPivotDimensions = map[string]bool{
	"monthly":         true,
	"quarterly":       true,
	"yearly":          true,
	"product":         true,
	"product_line":    true,
	"productLine":     true,
	"location":        true,
	"location_area":   true,
	"locationArea":    true,
	"client":          true,
	"clientCategory":  true,
	"client_category": true,
}

// normalizeDimension converts camelCase dimension keys to snake_case for SQL switch matching.
func normalizeDimension(dim string) string {
	switch dim {
	case "productLine":
		return "product_line"
	case "locationArea":
		return "location_area"
	case "clientCategory":
		return "client_category"
	default:
		return dim
	}
}

// getPivotDimensionConfig returns SQL fragments for the requested dimension.
// Dialect differences from postgres gold standard:
//   - date_trunc → DATE_FORMAT for period dimensions
//   - TO_CHAR → DATE_FORMAT
//   - EXTRACT(QUARTER...) → QUARTER()
//   - EXTRACT(YEAR...) → YEAR()
//   - ::text / ::int casts removed
func getPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		expr := "DATE_FORMAT(r.revenue_date, '%Y-%m-01')"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("DATE_FORMAT(%s, '%%M %%Y')", expr),
			selectID:  expr,
			groupBy:   expr,
		}
	case "quarterly":
		return pivotDimensionConfig{
			selectKey: "CONCAT('Q', QUARTER(r.revenue_date), ' ', YEAR(r.revenue_date))",
			selectID:  "CONCAT(YEAR(r.revenue_date), '-Q', QUARTER(r.revenue_date))",
			groupBy:   "YEAR(r.revenue_date), QUARTER(r.revenue_date)",
		}
	case "yearly":
		return pivotDimensionConfig{
			selectKey: "CAST(YEAR(r.revenue_date) AS CHAR)",
			selectID:  "CAST(YEAR(r.revenue_date) AS CHAR)",
			groupBy:   "YEAR(r.revenue_date)",
		}
	case "product":
		return pivotDimensionConfig{
			selectKey: "COALESCE(p.name, 'Unassigned')",
			selectID:  "COALESCE(rli.product_id, '__none__')",
			groupBy:   "rli.product_id, p.name",
		}
	case "product_line":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(pl.name, 'Unassigned')",
			selectID:   "COALESCE(p.line_id, '__none__')",
			groupBy:    "p.line_id, pl.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s pl ON pl.id = p.line_id", tc.Line),
		}
	case "location":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(l.name, 'Unassigned')",
			selectID:   "COALESCE(r.location_id, '__none__')",
			groupBy:    "r.location_id, l.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = r.location_id", tc.Location),
		}
	case "location_area":
		return pivotDimensionConfig{
			selectKey: "COALESCE(la.name, 'Unassigned')",
			selectID:  "COALESCE(l.location_area_id, '__none__')",
			groupBy:   "l.location_area_id, la.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s l ON l.id = r.location_id LEFT JOIN %s la ON la.id = l.location_area_id",
				tc.Location, tc.LocationArea),
		}
	case "client":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(cl.name, r.name, 'Unassigned')",
			selectID:   "COALESCE(r.client_id, '__none__')",
			groupBy:    "r.client_id, cl.name, r.name",
			extraJoins: "LEFT JOIN client cl ON cl.id = r.client_id",
		}
	case "client_category":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(cat.name, 'Unassigned')",
			selectID:   "COALESCE(cc.category_id, '__none__')",
			groupBy:    "cc.category_id, cat.name",
			extraJoins: "LEFT JOIN client cl ON cl.id = r.client_id LEFT JOIN client_category cc ON cc.id = cl.category_id LEFT JOIN category cat ON cat.id = cc.category_id",
		}
	default:
		return getPivotDimensionConfig(tc, "product")
	}
}

// buildRevenueReportQuery constructs the pivot SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries (?).
//
// Dialect differences from postgres gold standard:
//   - $N → ? (same left-to-right order)
//   - $N::timestamptz IS NULL → ? IS NULL
//   - date_trunc / TO_CHAR / EXTRACT → MySQL equivalents (see getPivotDimensionConfig)
//   - ::bigint / ::text casts removed
//   - active = true → active = 1
func buildRevenueReportQuery(tc TableConfig, req *revreportpb.RevenueReportRequest, workspaceID string) (string, []any) {
	primaryDim := normalizeDimension(req.GetPrimaryDimension())
	if !validPivotDimensions[primaryDim] {
		primaryDim = "monthly"
	}
	rowDim := normalizeDimension(req.GetRowDimension())
	if !validPivotDimensions[rowDim] {
		rowDim = "product"
	}

	colConfig := getPivotDimensionConfig(tc, primaryDim)
	rowConfig := getPivotDimensionConfig(tc, rowDim)

	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Args in same order as postgres gold standard:
	// ? = start_date, ? = end_date, ? = product_id, ? = location_id,
	// ? = revenue_category_id, ? = client_id (nil), ? = workspace_id
	baseArgs := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetProductId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetRevenueCategoryId()),
		nil, // client_id filter (not yet exposed in proto)
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH revenue_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(rli.total_price) AS total_revenue,
        COUNT(DISTINCT r.id)      AS transaction_count,
        SUM(rli.quantity)         AS total_quantity
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = 1
      AND r.status != 'cancelled'
      AND (? IS NULL OR r.revenue_date >= ?)
      AND (? IS NULL OR r.revenue_date <= ?)
      AND (? IS NULL OR rli.product_id = ?)
      AND (? IS NULL OR r.location_id = ?)
      AND (? IS NULL OR r.revenue_category_id = ?)
      AND (? IS NULL OR r.client_id = ?)
      AND (? IS NULL OR r.workspace_id = ?)
    GROUP BY %s, %s
)
SELECT row_key, row_id, col_key, col_id,
       total_revenue, transaction_count, total_quantity
FROM revenue_pivot
ORDER BY row_key, col_key`,
		rowConfig.selectKey, rowConfig.selectID,
		colConfig.selectKey, colConfig.selectID,
		tc.RevenueLineItem,
		tc.Revenue,
		tc.Product,
		extraJoins,
		rowConfig.groupBy, colConfig.groupBy,
	)

	// Expand each (? IS NULL OR col = ?) pair.
	args := []any{
		baseArgs[0], baseArgs[0], // start_date
		baseArgs[1], baseArgs[1], // end_date
		baseArgs[2], baseArgs[2], // product_id
		baseArgs[3], baseArgs[3], // location_id
		baseArgs[4], baseArgs[4], // category_id
		baseArgs[5], baseArgs[5], // client_id
		baseArgs[6], baseArgs[6], // workspace_id
	}

	return query, args
}

// mergeJoins combines two JOIN strings, deduplicating shared table aliases.
func mergeJoins(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a == b {
		return a
	}
	if strings.Contains(a, " l ON l.id") && strings.Contains(b, " l ON l.id") {
		parts := strings.SplitN(b, "LEFT JOIN", 3)
		if len(parts) == 3 {
			return a + " LEFT JOIN" + parts[2]
		}
	}
	if strings.Contains(a, " cl ON cl.id") && strings.Contains(b, " cl ON cl.id") {
		parts := strings.SplitN(b, "LEFT JOIN", 3)
		if len(parts) == 3 {
			return a + " LEFT JOIN" + parts[2]
		}
	}
	return a + " " + b
}

// nilIfEmpty returns nil for empty strings so they bind as SQL NULL.
// (Defined in adapter.go; re-declared here only if needed — already shared via package scope.)

// flatRow holds one database result row before pivoting.
type flatRow struct {
	RowKey           string
	RowID            string
	ColKey           string
	ColID            string
	TotalRevenue     int64
	TransactionCount int64
	TotalQuantity    float64
}

// pivotFlatRows transforms flat SQL result rows into the proto pivot response.
func pivotFlatRows(flat []flatRow, req *revreportpb.RevenueReportRequest) *revreportpb.RevenueReportResponse {
	if len(flat) == 0 {
		return &revreportpb.RevenueReportResponse{
			Success: true,
			Summary: buildRevenueSummary(nil, nil, req),
		}
	}

	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*revreportpb.RevenueReportCell)

	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*revreportpb.RevenueReportCell
	}
	rowOrder := make([]string, 0)
	rowSeen := make(map[string]bool)
	rowAccums := make(map[string]*rowAccum)

	for _, fr := range flat {
		if !colSeen[fr.ColKey] {
			colSeen[fr.ColKey] = true
			colOrder = append(colOrder, fr.ColKey)
			colID := fr.ColID
			colTotals[fr.ColKey] = &revreportpb.RevenueReportCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}
		if !rowSeen[fr.RowKey] {
			rowSeen[fr.RowKey] = true
			rowOrder = append(rowOrder, fr.RowKey)
			rowAccums[fr.RowKey] = &rowAccum{
				rowKey: fr.RowKey,
				rowID:  fr.RowID,
				cells:  make(map[string]*revreportpb.RevenueReportCell),
			}
		}
		ra := rowAccums[fr.RowKey]
		if _, ok := ra.cells[fr.ColKey]; !ok {
			colID := fr.ColID
			ra.cells[fr.ColKey] = &revreportpb.RevenueReportCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}
		cell := ra.cells[fr.ColKey]
		cell.TotalRevenue += fr.TotalRevenue
		cell.TransactionCount += fr.TransactionCount
		cell.TotalQuantity += fr.TotalQuantity

		ct := colTotals[fr.ColKey]
		ct.TotalRevenue += fr.TotalRevenue
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += fr.TotalQuantity
	}

	primaryDim := normalizeDimension(req.GetPrimaryDimension())
	isPeriodDim := primaryDim == "monthly" || primaryDim == "quarterly" || primaryDim == "yearly"
	if isPeriodDim {
		sort.Slice(colOrder, func(i, j int) bool {
			return colTotals[colOrder[i]].GetColumnId() > colTotals[colOrder[j]].GetColumnId()
		})
	} else {
		sort.Slice(colOrder, func(i, j int) bool {
			return strings.ToLower(colOrder[i]) < strings.ToLower(colOrder[j])
		})
	}

	colHeaders := make([]string, 0, len(colOrder))
	for _, ck := range colOrder {
		colHeaders = append(colHeaders, ck)
	}
	columnTotals := make([]*revreportpb.RevenueReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	rowDimNorm := normalizeDimension(req.GetRowDimension())
	isRowPeriod := rowDimNorm == "monthly" || rowDimNorm == "quarterly" || rowDimNorm == "yearly"
	if isRowPeriod {
		sort.Slice(rowOrder, func(i, j int) bool {
			return rowAccums[rowOrder[i]].rowID > rowAccums[rowOrder[j]].rowID
		})
	} else {
		sort.Slice(rowOrder, func(i, j int) bool {
			return strings.ToLower(rowOrder[i]) < strings.ToLower(rowOrder[j])
		})
	}

	pbRows := make([]*revreportpb.RevenueReportRow, 0, len(rowOrder))
	for _, rk := range rowOrder {
		ra := rowAccums[rk]
		cells := make([]*revreportpb.RevenueReportCell, 0, len(colOrder))
		var rowTotal int64
		var rowTxCount int64
		var rowQty float64
		for _, ck := range colOrder {
			if cell, ok := ra.cells[ck]; ok {
				cells = append(cells, cell)
				rowTotal += cell.TotalRevenue
				rowTxCount += cell.TransactionCount
				rowQty += cell.TotalQuantity
			} else {
				colID := colTotals[ck].GetColumnId()
				cells = append(cells, &revreportpb.RevenueReportCell{
					ColumnKey: ck,
					ColumnId:  &colID,
				})
			}
		}
		rowID := ra.rowID
		pbRows = append(pbRows, &revreportpb.RevenueReportRow{
			RowKey:              ra.rowKey,
			RowId:               &rowID,
			Cells:               cells,
			RowTotal:            rowTotal,
			RowTransactionCount: rowTxCount,
			RowTotalQuantity:    rowQty,
		})
	}

	summary := buildRevenueSummary(pbRows, columnTotals, req)
	return &revreportpb.RevenueReportResponse{
		ColumnKeys: colHeaders,
		Rows:       pbRows,
		Summary:    summary,
		Success:    true,
	}
}

// buildRevenueSummary computes report-level totals from assembled rows.
func buildRevenueSummary(rows []*revreportpb.RevenueReportRow, columnTotals []*revreportpb.RevenueReportCell, req *revreportpb.RevenueReportRequest) *revreportpb.RevenueReportSummary {
	s := &revreportpb.RevenueReportSummary{
		ColumnTotals: columnTotals,
	}
	for _, row := range rows {
		s.GrandTotal += row.RowTotal
		s.TotalTransactions += row.RowTransactionCount
		s.TotalQuantity += row.RowTotalQuantity
	}
	if req != nil {
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
