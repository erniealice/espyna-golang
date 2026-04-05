package ledger

import (
	"fmt"
	"sort"
	"strings"

	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
)

// validDisbursementPivotDimensions is a whitelist of allowed dimension values for disbursement reports.
var validDisbursementPivotDimensions = map[string]bool{
	"monthly":                true,
	"quarterly":              true,
	"yearly":                 true,
	"supplier":               true,
	"supplier_category":      true,
	"supplierCategory":       true,
	"location":               true,
	"location_area":          true,
	"locationArea":           true,
	"expenditure_category":   true,
	"expenditureCategory":    true,
	"disbursement_type":      true,
	"disbursementType":       true,
	"disbursement_method":    true,
	"disbursementMethod":     true,
}

// normalizeDisbursementDimension converts camelCase dimension keys to snake_case for SQL switch matching.
func normalizeDisbursementDimension(dim string) string {
	switch dim {
	case "supplierCategory":
		return "supplier_category"
	case "locationArea":
		return "location_area"
	case "expenditureCategory":
		return "expenditure_category"
	case "disbursementType":
		return "disbursement_type"
	case "disbursementMethod":
		return "disbursement_method"
	default:
		return dim
	}
}

// getDisbursementPivotDimensionConfig returns SQL fragments for the requested disbursement dimension.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
func getDisbursementPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	dateExpr := "TO_TIMESTAMP(d.payment_date / 1000.0)"

	switch dimension {
	case "monthly":
		expr := fmt.Sprintf("date_trunc('month', %s)", dateExpr)
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("TO_CHAR(%s, 'Month YYYY')", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "quarterly":
		expr := fmt.Sprintf("date_trunc('quarter', %s)", dateExpr)
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("'Q' || EXTRACT(QUARTER FROM %s)::int || ' ' || EXTRACT(YEAR FROM %s)::int", expr, expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "yearly":
		expr := fmt.Sprintf("date_trunc('year', %s)", dateExpr)
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("EXTRACT(YEAR FROM %s)::int::text", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "supplier":
		return pivotDimensionConfig{
			selectKey: "COALESCE(s.company_name, 'Unknown')",
			selectID:  "COALESCE(e.supplier_id, '__none__')",
			groupBy:   "e.supplier_id, s.company_name",
		}
	case "supplier_category":
		return pivotDimensionConfig{
			selectKey: "COALESCE(sc.name, 'Unassigned')",
			selectID:  "COALESCE(s.category_id, '__none__')",
			groupBy:   "s.category_id, sc.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s sc ON sc.id = s.category_id",
				tc.SupplierCategory),
		}
	case "location":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(l.name, 'Unassigned')",
			selectID:   "COALESCE(e.location_id, '__none__')",
			groupBy:    "e.location_id, l.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s l ON l.id = e.location_id", tc.Location),
		}
	case "location_area":
		return pivotDimensionConfig{
			selectKey: "COALESCE(la.name, 'Unassigned')",
			selectID:  "COALESCE(l.location_area_id, '__none__')",
			groupBy:   "l.location_area_id, la.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s l ON l.id = e.location_id LEFT JOIN %s la ON la.id = l.location_area_id",
				tc.Location, tc.LocationArea),
		}
	case "expenditure_category":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(ec.name, 'Unassigned')",
			selectID:   "COALESCE(e.expenditure_category_id, '__none__')",
			groupBy:    "e.expenditure_category_id, ec.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s ec ON ec.id = e.expenditure_category_id", tc.ExpenditureCategory),
		}
	case "disbursement_type":
		return pivotDimensionConfig{
			selectKey: "COALESCE(d.disbursement_type, 'Unassigned')",
			selectID:  "COALESCE(d.disbursement_type, '__none__')",
			groupBy:   "d.disbursement_type",
		}
	case "disbursement_method":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(dm.name, 'Unassigned')",
			selectID:   "COALESCE(d.disbursement_method_id, '__none__')",
			groupBy:    "d.disbursement_method_id, dm.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s dm ON dm.id = d.disbursement_method_id", tc.DisbursementMethod),
		}
	default:
		return getDisbursementPivotDimensionConfig(tc, "supplier")
	}
}

// buildDisbursementReportQuery constructs the pivot SQL query and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries ($1, $2, ...).
//
// The query groups disbursements by two independent dimensions:
//   - rowDimension     -> each output row
//   - primaryDimension -> each column within a row (the pivot axis)
func buildDisbursementReportQuery(tc TableConfig, req *disbreportpb.DisbursementReportRequest) (string, []any) {
	// Validate and normalise dimensions.
	primaryDim := normalizeDisbursementDimension(req.GetPrimaryDimension())
	if !validDisbursementPivotDimensions[primaryDim] {
		primaryDim = "monthly"
	}
	rowDim := normalizeDisbursementDimension(req.GetRowDimension())
	if !validDisbursementPivotDimensions[rowDim] {
		rowDim = "supplier"
	}

	colConfig := getDisbursementPivotDimensionConfig(tc, primaryDim)
	rowConfig := getDisbursementPivotDimensionConfig(tc, rowDim)

	// Combine extra JOINs from both dimensions, deduplicating shared table aliases.
	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Build parameter list.
	// $1 = start_date (timestamptz or NULL)
	// $2 = end_date   (timestamptz or NULL)
	// $3 = supplier_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = expenditure_category_id (text or NULL)
	// $6 = disbursement_type (text or NULL)
	// $7 = disbursement_method_id (text or NULL)
	// $8 = supplier_category_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetSupplierId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetExpenditureCategoryId()),
		nilIfEmpty(req.GetDisbursementType()),
		nilIfEmpty(req.GetDisbursementMethodId()),
		nilIfEmpty(req.GetSupplierCategoryId()),
	}

	dateExpr := "TO_TIMESTAMP(d.payment_date / 1000.0)"

	query := fmt.Sprintf(`
WITH disbursement_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(d.amount)::bigint        AS total_disbursement,
        COUNT(DISTINCT d.id)         AS transaction_count,
        COUNT(DISTINCT d.id)::bigint AS total_quantity
    FROM %s d
    JOIN %s e ON e.id = d.expenditure_id
    LEFT JOIN %s s ON s.id = e.supplier_id
    %s
    WHERE d.active = true
      AND d.status IN ('paid', 'completed')
      AND ($1::timestamptz IS NULL OR %s >= $1::timestamptz)
      AND ($2::timestamptz IS NULL OR %s <= $2::timestamptz)
      AND ($3::text IS NULL OR e.supplier_id = $3)
      AND ($4::text IS NULL OR e.location_id = $4)
      AND ($5::text IS NULL OR e.expenditure_category_id = $5)
      AND ($6::text IS NULL OR d.disbursement_type = $6)
      AND ($7::text IS NULL OR d.disbursement_method_id = $7)
      AND ($8::text IS NULL OR s.category_id = $8)
    GROUP BY %s, %s
)
SELECT row_key, row_id, col_key, col_id,
       total_disbursement, transaction_count, total_quantity
FROM disbursement_pivot
ORDER BY row_key, col_key`,
		rowConfig.selectKey, rowConfig.selectID,
		colConfig.selectKey, colConfig.selectID,
		tc.TreasuryDisbursement,
		tc.Expenditure,
		tc.Supplier,
		extraJoins,
		dateExpr, dateExpr,
		rowConfig.groupBy, colConfig.groupBy,
	)

	return query, args
}

// disbursementFlatRow holds one database result row before pivoting.
type disbursementFlatRow struct {
	RowKey             string
	RowID              string
	ColKey             string
	ColID              string
	TotalDisbursement  int64
	TransactionCount   int64
	TotalQuantity      int64
}

// pivotFlatDisbursementRows transforms flat SQL result rows into the proto pivot response.
// It groups rows by row_key, builds one DisbursementReportCell per column, and
// computes row totals and report-level summary.
func pivotFlatDisbursementRows(flat []disbursementFlatRow, req *disbreportpb.DisbursementReportRequest) *disbreportpb.DisbursementReportResponse {
	if len(flat) == 0 {
		return &disbreportpb.DisbursementReportResponse{
			Success: true,
			Summary: buildDisbursementSummary(nil, nil, req),
		}
	}

	// Track ordered column keys and column-level totals.
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*disbreportpb.DisbursementReportCell)

	// Group flat rows by row_key (preserving insertion order).
	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*disbreportpb.DisbursementReportCell // colKey -> cell
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
			colTotals[fr.ColKey] = &disbreportpb.DisbursementReportCell{
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
				cells:  make(map[string]*disbreportpb.DisbursementReportCell),
			}
		}

		ra := rowAccums[fr.RowKey]
		if _, ok := ra.cells[fr.ColKey]; !ok {
			colID := fr.ColID
			ra.cells[fr.ColKey] = &disbreportpb.DisbursementReportCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}
		cell := ra.cells[fr.ColKey]
		cell.TotalDisbursement += fr.TotalDisbursement
		cell.TransactionCount += fr.TransactionCount
		cell.TotalQuantity += float64(fr.TotalQuantity)

		// Accumulate column totals.
		ct := colTotals[fr.ColKey]
		ct.TotalDisbursement += fr.TotalDisbursement
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += float64(fr.TotalQuantity)
	}

	// Sort columns: periods descending (latest first), entities alphabetical.
	primaryDim := normalizeDisbursementDimension(req.GetPrimaryDimension())
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
	columnTotals := make([]*disbreportpb.DisbursementReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	// Sort rows: periods descending (latest first), entities alphabetical.
	rowDimNorm := normalizeDisbursementDimension(req.GetRowDimension())
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
	pbRows := make([]*disbreportpb.DisbursementReportRow, 0, len(rowOrder))
	for _, rk := range rowOrder {
		ra := rowAccums[rk]
		cells := make([]*disbreportpb.DisbursementReportCell, 0, len(colOrder))
		var rowTotal int64
		var rowTxCount int64
		var rowQty float64
		for _, ck := range colOrder {
			if cell, ok := ra.cells[ck]; ok {
				cells = append(cells, cell)
				rowTotal += cell.TotalDisbursement
				rowTxCount += cell.TransactionCount
				rowQty += cell.TotalQuantity
			} else {
				// Emit a zero cell so columns stay aligned.
				colID := colTotals[ck].GetColumnId()
				cells = append(cells, &disbreportpb.DisbursementReportCell{
					ColumnKey: ck,
					ColumnId:  &colID,
				})
			}
		}
		rowID := ra.rowID
		pbRows = append(pbRows, &disbreportpb.DisbursementReportRow{
			RowKey:              ra.rowKey,
			RowId:               &rowID,
			Cells:               cells,
			RowTotal:            rowTotal,
			RowTransactionCount: rowTxCount,
			RowTotalQuantity:    rowQty,
		})
	}

	summary := buildDisbursementSummary(pbRows, columnTotals, req)

	return &disbreportpb.DisbursementReportResponse{
		ColumnKeys: colHeaders,
		Rows:       pbRows,
		Summary:    summary,
		Success:    true,
	}
}

// buildDisbursementSummary computes report-level totals from assembled rows.
func buildDisbursementSummary(rows []*disbreportpb.DisbursementReportRow, columnTotals []*disbreportpb.DisbursementReportCell, req *disbreportpb.DisbursementReportRequest) *disbreportpb.DisbursementReportSummary {
	s := &disbreportpb.DisbursementReportSummary{
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
