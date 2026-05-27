//go:build sqlserver

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

// normalizeDimension converts camelCase dimension keys to snake_case.
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
// SQL Server period expressions use DATEFROMPARTS / FORMAT instead of date_trunc / TO_CHAR.
func getPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		// Month bucket: DATEFROMPARTS(YEAR(col), MONTH(col), 1)
		expr := "DATEFROMPARTS(YEAR(r.revenue_date), MONTH(r.revenue_date), 1)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("FORMAT(%s, 'MMMM yyyy')", expr),
			selectID:  fmt.Sprintf("CAST(%s AS nvarchar(30))", expr),
			groupBy:   expr,
		}
	case "quarterly":
		// Approximate quarter bucket with first month of quarter.
		expr := "DATEFROMPARTS(YEAR(r.revenue_date), ((MONTH(r.revenue_date) - 1) / 3) * 3 + 1, 1)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("'Q' + CAST(DATEPART(quarter, r.revenue_date) AS nvarchar) + ' ' + CAST(YEAR(r.revenue_date) AS nvarchar)"),
			selectID:  fmt.Sprintf("CAST(%s AS nvarchar(30))", expr),
			groupBy:   "YEAR(r.revenue_date), DATEPART(quarter, r.revenue_date)",
		}
	case "yearly":
		expr := "DATEFROMPARTS(YEAR(r.revenue_date), 1, 1)"
		return pivotDimensionConfig{
			selectKey: "CAST(YEAR(r.revenue_date) AS nvarchar(10))",
			selectID:  fmt.Sprintf("CAST(%s AS nvarchar(30))", expr),
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
// All user-provided values use parameterized queries (@p1, @p2, ...).
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - ::timestamptz IS NULL → @pN IS NULL.
//   - date_trunc / TO_CHAR → DATEFROMPARTS / FORMAT.
//   - ::bigint → CAST(… AS bigint).
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

	// @p1 = start_date (datetime2 or NULL)
	// @p2 = end_date   (datetime2 or NULL)
	// @p3 = product_id (text or NULL)
	// @p4 = location_id (text or NULL)
	// @p5 = revenue_category_id (text or NULL)
	// @p6 = client_id (text or NULL)
	// @p7 = workspace_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetProductId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetRevenueCategoryId()),
		nil, // @p6: client_id filter (not yet exposed in proto)
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH revenue_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        CAST(SUM(rli.total_price) AS bigint) AS total_revenue,
        COUNT(DISTINCT r.id)                 AS transaction_count,
        CAST(SUM(rli.quantity) AS bigint)    AS total_quantity
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = 1
      AND r.status != 'cancelled'
      AND (@p1 IS NULL OR r.revenue_date >= @p1)
      AND (@p2 IS NULL OR r.revenue_date <= @p2)
      AND (@p3 IS NULL OR rli.product_id = @p3)
      AND (@p4 IS NULL OR r.location_id = @p4)
      AND (@p5 IS NULL OR r.revenue_category_id = @p5)
      AND (@p6 IS NULL OR r.client_id = @p6)
      AND (@p7 IS NULL OR r.workspace_id = @p7)
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

	return query, args
}

// mergeJoins combines two JOIN strings, deduplicating the location table alias "l"
// when both dimensions require it.
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
