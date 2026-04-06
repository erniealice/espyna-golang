package ledger

import (
	"fmt"
	"sort"
	"strings"

	revreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
)

// pivotDimensionConfig defines SQL fragments for one axis of the pivot.
type pivotDimensionConfig struct {
	selectKey  string // SQL expression yielding a human-readable label
	selectID   string // SQL expression yielding an entity ID
	groupBy    string // GROUP BY fragment (may be multiple columns)
	extraJoins string // Additional JOINs required by this dimension
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
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
func getPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		expr := "date_trunc('month', r.revenue_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("TO_CHAR(%s, 'Month YYYY')", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "quarterly":
		expr := "date_trunc('quarter', r.revenue_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("'Q' || EXTRACT(QUARTER FROM %s)::int || ' ' || EXTRACT(YEAR FROM %s)::int", expr, expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "yearly":
		expr := "date_trunc('year', r.revenue_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("EXTRACT(YEAR FROM %s)::int::text", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "product":
		return pivotDimensionConfig{
			selectKey: "COALESCE(p.name, 'Unassigned')",
			selectID:  "COALESCE(rli.product_id, '__none__')",
			groupBy:   "rli.product_id, p.name",
		}
	case "product_line":
		return pivotDimensionConfig{
			selectKey: "COALESCE(c.name, 'Unassigned')",
			selectID:  "COALESCE(pc_first.collection_id, '__none__')",
			groupBy:   "pc_first.collection_id, c.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN LATERAL (SELECT collection_id FROM %s WHERE product_id = rli.product_id AND active = true ORDER BY sort_order LIMIT 1) pc_first ON true"+
					" LEFT JOIN %s c ON c.id = pc_first.collection_id",
				tc.ProductCollection, tc.Collection),
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
// All user-provided values use parameterized queries ($1, $2, ...).
//
// The query groups revenue line items by two independent dimensions:
//   - rowDimension  → each output row
//   - primaryDimension → each column within a row (the pivot axis)
func buildRevenueReportQuery(tc TableConfig, req *revreportpb.RevenueReportRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimensions.
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

	// Combine extra JOINs from both dimensions, deduplicating the location join
	// when both dimensions need it (e.g. location + location_area).
	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Build parameter list.
	// $1 = start_date (timestamptz or NULL)
	// $2 = end_date   (timestamptz or NULL)
	// $3 = product_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = revenue_category_id (text or NULL)
	// $6 = client_id (text or NULL)
	// $7 = workspace_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetProductId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetRevenueCategoryId()),
		nil, // $6: client_id filter (not yet exposed in proto)
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH revenue_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(rli.total_price)::bigint AS total_revenue,
        COUNT(DISTINCT r.id)         AS transaction_count,
        SUM(rli.quantity)::bigint    AS total_quantity
    FROM %s rli
    JOIN %s r ON r.id = rli.revenue_id
    LEFT JOIN %s p ON p.id = rli.product_id
    %s
    WHERE r.active = true
      AND r.status != 'cancelled'
      AND ($1::timestamptz IS NULL OR r.revenue_date::timestamptz >= $1::timestamptz)
      AND ($2::timestamptz IS NULL OR r.revenue_date::timestamptz <= $2::timestamptz)
      AND ($3::text IS NULL OR rli.product_id = $3)
      AND ($4::text IS NULL OR r.location_id = $4)
      AND ($5::text IS NULL OR r.revenue_category_id = $5)
      AND ($6::text IS NULL OR r.client_id = $6)
      AND ($7::text IS NULL OR r.workspace_id = $7)
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
// when both dimensions require it (e.g. location + location_area).
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
	// If both sides join the location table (alias "l"), only include it once.
	// For location_area, b has "LEFT JOIN {loc} l ON... LEFT JOIN {la} la ON..."
	// a already has "LEFT JOIN {loc} l ON..." so we strip it from b and keep
	// only the remaining "LEFT JOIN {la} la ON..." part.
	if strings.Contains(a, " l ON l.id") && strings.Contains(b, " l ON l.id") {
		parts := strings.SplitN(b, "LEFT JOIN", 3)
		if len(parts) == 3 {
			return a + " LEFT JOIN" + parts[2]
		}
	}
	// If both sides join the client table (alias "cl"), only include it once.
	// For client_category, b has "LEFT JOIN client cl ON... LEFT JOIN category cat ON..."
	// a already has "LEFT JOIN client cl ON..." so we strip it from b and keep
	// only the remaining "LEFT JOIN category cat ON..." part.
	if strings.Contains(a, " cl ON cl.id") && strings.Contains(b, " cl ON cl.id") {
		parts := strings.SplitN(b, "LEFT JOIN", 3)
		if len(parts) == 3 {
			return a + " LEFT JOIN" + parts[2]
		}
	}
	return a + " " + b
}

// nilIfEmpty returns nil for empty strings so they bind as SQL NULL.
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
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
// It groups rows by row_key, builds one RevenueReportCell per column, and
// computes row totals and report-level summary.
func pivotFlatRows(flat []flatRow, req *revreportpb.RevenueReportRequest) *revreportpb.RevenueReportResponse {
	if len(flat) == 0 {
		return &revreportpb.RevenueReportResponse{
			Success: true,
			Summary: buildRevenueSummary(nil, nil, req),
		}
	}

	// Track ordered column keys and column-level totals.
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*revreportpb.RevenueReportCell)

	// Group flat rows by row_key (preserving insertion order).
	type rowAccum struct {
		rowKey  string
		rowID   string
		cells   map[string]*revreportpb.RevenueReportCell // colKey → cell
	}
	rowOrder := make([]string, 0)
	rowSeen := make(map[string]bool)
	rowAccums := make(map[string]*rowAccum)

	for _, fr := range flat {
		// Track column order.
		if !colSeen[fr.ColKey] {
			colSeen[fr.ColKey] = true
			colOrder = append(colOrder, fr.ColKey)
			colID := fr.ColID
			colTotals[fr.ColKey] = &revreportpb.RevenueReportCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}

		// Track row order.
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

		// Accumulate column totals.
		ct := colTotals[fr.ColKey]
		ct.TotalRevenue += fr.TotalRevenue
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += fr.TotalQuantity
	}

	// Sort columns: periods descending (latest first), entities alphabetical.
	primaryDim := normalizeDimension(req.GetPrimaryDimension())
	isPeriodDim := primaryDim == "monthly" || primaryDim == "quarterly" || primaryDim == "yearly"
	if isPeriodDim {
		// Period columns have colID as a timestamp string — sort descending (latest first).
		sort.Slice(colOrder, func(i, j int) bool {
			idI := colTotals[colOrder[i]].GetColumnId()
			idJ := colTotals[colOrder[j]].GetColumnId()
			return idI > idJ // descending
		})
	} else {
		// Entity columns (product, location) — sort alphabetically by display key.
		sort.Slice(colOrder, func(i, j int) bool {
			return strings.ToLower(colOrder[i]) < strings.ToLower(colOrder[j])
		})
	}

	// Build column header list (ordered) — proto field is repeated string.
	colHeaders := make([]string, 0, len(colOrder))
	for _, ck := range colOrder {
		colHeaders = append(colHeaders, ck)
	}

	// Build ordered column totals list.
	columnTotals := make([]*revreportpb.RevenueReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	// Sort rows: periods descending (latest first), entities alphabetical.
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

	// Build rows.
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
				// Emit a zero cell so columns stay aligned.
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
