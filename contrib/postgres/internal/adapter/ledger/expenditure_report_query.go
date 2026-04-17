//go:build postgresql

package ledger

import (
	"fmt"
	"sort"
	"strings"

	expreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
)

// validExpenditurePivotDimensions is a whitelist of allowed dimension values for expenditure reports.
var validExpenditurePivotDimensions = map[string]bool{
	"monthly":            true,
	"quarterly":          true,
	"yearly":             true,
	"product":            true,
	"product_line":       true,
	"productLine":        true,
	"location":           true,
	"location_area":      true,
	"locationArea":       true,
	"category":           true,
	"supplier":           true,
	"expenditure_type":   true,
	"expenditureType":    true,
	"supplierCategory":   true,
	"supplier_category":  true,
}

// normalizeExpenditureDimension converts camelCase dimension keys to snake_case for SQL switch matching.
func normalizeExpenditureDimension(dim string) string {
	switch dim {
	case "productLine":
		return "product_line"
	case "locationArea":
		return "location_area"
	case "expenditureType":
		return "expenditure_type"
	case "supplierCategory":
		return "supplier_category"
	default:
		return dim
	}
}

// getExpenditurePivotDimensionConfig returns SQL fragments for the requested expenditure dimension.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
func getExpenditurePivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		expr := "date_trunc('month', e.expenditure_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("TO_CHAR(%s, 'Month YYYY')", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "quarterly":
		expr := "date_trunc('quarter', e.expenditure_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("'Q' || EXTRACT(QUARTER FROM %s)::int || ' ' || EXTRACT(YEAR FROM %s)::int", expr, expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "yearly":
		expr := "date_trunc('year', e.expenditure_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("EXTRACT(YEAR FROM %s)::int::text", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "product":
		return pivotDimensionConfig{
			selectKey: "COALESCE(p.name, 'Unassigned')",
			selectID:  "COALESCE(eli.product_id, '__none__')",
			groupBy:   "eli.product_id, p.name",
		}
	case "product_line":
		// Uses product.line_id (one-to-many FK) for unambiguous line attribution.
		return pivotDimensionConfig{
			selectKey:  "COALESCE(pl.name, 'Unassigned')",
			selectID:   "COALESCE(p.line_id, '__none__')",
			groupBy:    "p.line_id, pl.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s pl ON pl.id = p.line_id", tc.Line),
		}
	case "location":
		return pivotDimensionConfig{
			selectKey: "COALESCE(l.name, 'Unassigned')",
			selectID:  "COALESCE(e.location_id, '__none__')",
			groupBy:   "e.location_id, l.name",
			// l is always joined in the base query; no extraJoins needed.
		}
	case "location_area":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(la.name, 'Unassigned')",
			selectID:   "COALESCE(l.location_area_id, '__none__')",
			groupBy:    "l.location_area_id, la.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s la ON la.id = l.location_area_id", tc.LocationArea),
		}
	case "category":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(ec.name, 'Unassigned')",
			selectID:   "COALESCE(e.expenditure_category_id, '__none__')",
			groupBy:    "e.expenditure_category_id, ec.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s ec ON ec.id = e.expenditure_category_id", tc.ExpenditureCategory),
		}
	case "supplier":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(s.company_name, 'No Supplier')",
			selectID:   "COALESCE(e.supplier_id, '__none__')",
			groupBy:    "e.supplier_id, s.company_name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s s ON s.id = e.supplier_id", tc.Supplier),
		}
	case "expenditure_type":
		return pivotDimensionConfig{
			selectKey: "COALESCE(e.expenditure_type, 'Unassigned')",
			selectID:  "COALESCE(e.expenditure_type, '__none__')",
			groupBy:   "e.expenditure_type",
		}
	case "supplier_category":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(sc.name, 'Unassigned')",
			selectID:   "COALESCE(s.category_id, '__none__')",
			groupBy:    "s.category_id, sc.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s s ON s.id = e.supplier_id LEFT JOIN %s sc ON sc.id = s.category_id", tc.Supplier, tc.SupplierCategory),
		}
	default:
		return getExpenditurePivotDimensionConfig(tc, "product")
	}
}

// buildExpenditureReportQuery constructs the pivot SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries ($1, $2, ...).
//
// The query groups expenditure line items by two independent dimensions:
//   - rowDimension  -> each output row
//   - primaryDimension -> each column within a row (the pivot axis)
func buildExpenditureReportQuery(tc TableConfig, req *expreportpb.ExpenditureReportRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimensions.
	primaryDim := normalizeExpenditureDimension(req.GetPrimaryDimension())
	if !validExpenditurePivotDimensions[primaryDim] {
		primaryDim = "monthly"
	}
	rowDim := normalizeExpenditureDimension(req.GetRowDimension())
	if !validExpenditurePivotDimensions[rowDim] {
		rowDim = "product"
	}

	colConfig := getExpenditurePivotDimensionConfig(tc, primaryDim)
	rowConfig := getExpenditurePivotDimensionConfig(tc, rowDim)

	// Combine extra JOINs from both dimensions, deduplicating shared table aliases.
	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Build parameter list.
	// $1 = start_date (timestamptz or NULL)
	// $2 = end_date   (timestamptz or NULL)
	// $3 = product_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = expenditure_category_id (text or NULL)
	// $6 = supplier_id (text or NULL)
	// $7 = expenditure_type (text or NULL)
	// $8 = location_area_id (text or NULL)
	// $9 = workspace_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetProductId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetExpenditureCategoryId()),
		nilIfEmpty(req.GetSupplierId()),
		nilIfEmpty(req.GetExpenditureType()),
		nilIfEmpty(req.GetLocationAreaId()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH expenditure_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(eli.line_amount)::bigint AS total_expenditure,
        COUNT(DISTINCT e.id)         AS transaction_count,
        SUM(eli.quantity)::bigint    AS total_quantity
    FROM %s eli
    JOIN %s e ON e.id = eli.expenditure_id
    LEFT JOIN %s p ON p.id = eli.product_id
    LEFT JOIN %s l ON l.id = e.location_id
    %s
    WHERE e.active = true
      AND e.status NOT IN ('cancelled', 'draft')
      AND ($1::timestamptz IS NULL OR e.expenditure_date::timestamptz >= $1::timestamptz)
      AND ($2::timestamptz IS NULL OR e.expenditure_date::timestamptz <= $2::timestamptz)
      AND ($3::text IS NULL OR eli.product_id = $3)
      AND ($4::text IS NULL OR e.location_id = $4)
      AND ($5::text IS NULL OR e.expenditure_category_id = $5)
      AND ($6::text IS NULL OR e.supplier_id = $6)
      AND ($7::text IS NULL OR e.expenditure_type = $7)
      AND ($8::text IS NULL OR l.location_area_id = $8)
      AND ($9::text IS NULL OR e.workspace_id = $9)
    GROUP BY %s, %s
)
SELECT row_key, row_id, col_key, col_id,
       total_expenditure, transaction_count, total_quantity
FROM expenditure_pivot
ORDER BY row_key, col_key`,
		rowConfig.selectKey, rowConfig.selectID,
		colConfig.selectKey, colConfig.selectID,
		tc.ExpenditureLineItem,
		tc.Expenditure,
		tc.Product,
		tc.Location,
		extraJoins,
		rowConfig.groupBy, colConfig.groupBy,
	)

	return query, args
}

// expenditureFlatRow holds one database result row before pivoting.
type expenditureFlatRow struct {
	RowKey            string
	RowID             string
	ColKey            string
	ColID             string
	TotalExpenditure  int64
	TransactionCount  int64
	TotalQuantity     float64
}

// pivotFlatExpenditureRows transforms flat SQL result rows into the proto pivot response.
// It groups rows by row_key, builds one ExpenditureReportCell per column, and
// computes row totals and report-level summary.
func pivotFlatExpenditureRows(flat []expenditureFlatRow, req *expreportpb.ExpenditureReportRequest) *expreportpb.ExpenditureReportResponse {
	if len(flat) == 0 {
		return &expreportpb.ExpenditureReportResponse{
			Success: true,
			Summary: buildExpenditureSummary(nil, nil, req),
		}
	}

	// Track ordered column keys and column-level totals.
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*expreportpb.ExpenditureReportCell)

	// Group flat rows by row_key (preserving insertion order).
	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*expreportpb.ExpenditureReportCell // colKey -> cell
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
			colTotals[fr.ColKey] = &expreportpb.ExpenditureReportCell{
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
				cells:  make(map[string]*expreportpb.ExpenditureReportCell),
			}
		}

		ra := rowAccums[fr.RowKey]
		if _, ok := ra.cells[fr.ColKey]; !ok {
			colID := fr.ColID
			ra.cells[fr.ColKey] = &expreportpb.ExpenditureReportCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}
		cell := ra.cells[fr.ColKey]
		cell.TotalExpenditure += fr.TotalExpenditure
		cell.TransactionCount += fr.TransactionCount
		cell.TotalQuantity += fr.TotalQuantity

		// Accumulate column totals.
		ct := colTotals[fr.ColKey]
		ct.TotalExpenditure += fr.TotalExpenditure
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += fr.TotalQuantity
	}

	// Sort columns: periods descending (latest first), entities alphabetical.
	primaryDim := normalizeExpenditureDimension(req.GetPrimaryDimension())
	isPeriodDim := primaryDim == "monthly" || primaryDim == "quarterly" || primaryDim == "yearly"
	if isPeriodDim {
		sort.Slice(colOrder, func(i, j int) bool {
			idI := colTotals[colOrder[i]].GetColumnId()
			idJ := colTotals[colOrder[j]].GetColumnId()
			return idI > idJ // descending
		})
	} else {
		sort.Slice(colOrder, func(i, j int) bool {
			return strings.ToLower(colOrder[i]) < strings.ToLower(colOrder[j])
		})
	}

	// Build column header list (ordered).
	colHeaders := make([]string, 0, len(colOrder))
	for _, ck := range colOrder {
		colHeaders = append(colHeaders, ck)
	}

	// Build ordered column totals list.
	columnTotals := make([]*expreportpb.ExpenditureReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	// Sort rows: periods descending (latest first), entities alphabetical.
	rowDimNorm := normalizeExpenditureDimension(req.GetRowDimension())
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
	pbRows := make([]*expreportpb.ExpenditureReportRow, 0, len(rowOrder))
	for _, rk := range rowOrder {
		ra := rowAccums[rk]
		cells := make([]*expreportpb.ExpenditureReportCell, 0, len(colOrder))
		var rowTotal int64
		var rowTxCount int64
		var rowQty float64
		for _, ck := range colOrder {
			if cell, ok := ra.cells[ck]; ok {
				cells = append(cells, cell)
				rowTotal += cell.TotalExpenditure
				rowTxCount += cell.TransactionCount
				rowQty += cell.TotalQuantity
			} else {
				// Emit a zero cell so columns stay aligned.
				colID := colTotals[ck].GetColumnId()
				cells = append(cells, &expreportpb.ExpenditureReportCell{
					ColumnKey: ck,
					ColumnId:  &colID,
				})
			}
		}
		rowID := ra.rowID
		pbRows = append(pbRows, &expreportpb.ExpenditureReportRow{
			RowKey:              ra.rowKey,
			RowId:               &rowID,
			Cells:               cells,
			RowTotal:            rowTotal,
			RowTransactionCount: rowTxCount,
			RowTotalQuantity:    rowQty,
		})
	}

	summary := buildExpenditureSummary(pbRows, columnTotals, req)

	return &expreportpb.ExpenditureReportResponse{
		ColumnKeys: colHeaders,
		Rows:       pbRows,
		Summary:    summary,
		Success:    true,
	}
}

// buildExpenditureSummary computes report-level totals from assembled rows.
func buildExpenditureSummary(rows []*expreportpb.ExpenditureReportRow, columnTotals []*expreportpb.ExpenditureReportCell, req *expreportpb.ExpenditureReportRequest) *expreportpb.ExpenditureReportSummary {
	s := &expreportpb.ExpenditureReportSummary{
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