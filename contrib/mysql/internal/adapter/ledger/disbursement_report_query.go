//go:build mysql

package ledger

import (
	"fmt"
	"sort"
	"strings"

	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
)

// validDisbursementPivotDimensions is a whitelist of allowed dimension values for disbursement reports.
var validDisbursementPivotDimensions = map[string]bool{
	"monthly":              true,
	"quarterly":            true,
	"yearly":               true,
	"supplier":             true,
	"supplier_category":    true,
	"supplierCategory":     true,
	"location":             true,
	"location_area":        true,
	"locationArea":         true,
	"expenditure_category": true,
	"expenditureCategory":  true,
	"disbursement_type":    true,
	"disbursementType":     true,
	"disbursement_method":  true,
	"disbursementMethod":   true,
}

// normalizeDisbursementDimension converts camelCase dimension keys to snake_case.
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
// Dialect: TO_TIMESTAMP(x/1000.0) → FROM_UNIXTIME(x/1000); date_trunc → DATE_FORMAT;
// EXTRACT → YEAR()/QUARTER(); ::text / ::int casts removed.
func getDisbursementPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	// MySQL: payment_date stored as epoch millis; convert to datetime.
	dateExpr := "FROM_UNIXTIME(d.payment_date / 1000)"

	switch dimension {
	case "monthly":
		expr := fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-01')", dateExpr)
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("DATE_FORMAT(%s, '%%M %%Y')", expr),
			selectID:  expr,
			groupBy:   expr,
		}
	case "quarterly":
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("CONCAT('Q', QUARTER(%s), ' ', YEAR(%s))", dateExpr, dateExpr),
			selectID:  fmt.Sprintf("CONCAT(YEAR(%s), '-Q', QUARTER(%s))", dateExpr, dateExpr),
			groupBy:   fmt.Sprintf("YEAR(%s), QUARTER(%s)", dateExpr, dateExpr),
		}
	case "yearly":
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("CAST(YEAR(%s) AS CHAR)", dateExpr),
			selectID:  fmt.Sprintf("CAST(YEAR(%s) AS CHAR)", dateExpr),
			groupBy:   fmt.Sprintf("YEAR(%s)", dateExpr),
		}
	case "supplier":
		return pivotDimensionConfig{
			selectKey: "COALESCE(s.name, 'Unknown')",
			selectID:  "COALESCE(e.supplier_id, '__none__')",
			groupBy:   "e.supplier_id, s.name",
		}
	case "supplier_category":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(sc.name, 'Unassigned')",
			selectID:   "COALESCE(s.category_id, '__none__')",
			groupBy:    "s.category_id, sc.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s sc ON sc.id = s.category_id", tc.SupplierCategory),
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
// Dialect: $N → ?; TO_TIMESTAMP → FROM_UNIXTIME; ::timestamptz IS NULL → ? IS NULL; active = true → active = 1.
func buildDisbursementReportQuery(tc TableConfig, req *disbreportpb.DisbursementReportRequest, workspaceID string) (string, []any) {
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

	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Args in same order as postgres gold standard (9 base args).
	baseArgs := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetSupplierId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetExpenditureCategoryId()),
		nilIfEmpty(req.GetDisbursementType()),
		nilIfEmpty(req.GetDisbursementMethodId()),
		nilIfEmpty(req.GetSupplierCategoryId()),
		nilIfEmpty(workspaceID),
	}

	// MySQL date expression for comparing payment_date (epoch millis) to a date filter string.
	// Postgres uses TO_TIMESTAMP(d.payment_date / 1000.0) >= $1::timestamptz
	// MySQL: FROM_UNIXTIME(d.payment_date / 1000) >= ?
	dateExpr := "FROM_UNIXTIME(d.payment_date / 1000)"

	query := fmt.Sprintf(`
WITH disbursement_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        SUM(d.amount)         AS total_disbursement,
        COUNT(DISTINCT d.id)  AS transaction_count,
        COUNT(DISTINCT d.id)  AS total_quantity
    FROM %s d
    JOIN %s e ON e.id = d.expenditure_id
    LEFT JOIN %s s ON s.id = e.supplier_id
    %s
    WHERE d.active = 1
      AND d.status IN ('paid', 'completed')
      AND (? IS NULL OR %s >= ?)
      AND (? IS NULL OR %s <= ?)
      AND (? IS NULL OR e.supplier_id = ?)
      AND (? IS NULL OR e.location_id = ?)
      AND (? IS NULL OR e.expenditure_category_id = ?)
      AND (? IS NULL OR d.disbursement_type = ?)
      AND (? IS NULL OR d.disbursement_method_id = ?)
      AND (? IS NULL OR s.category_id = ?)
      AND (? IS NULL OR e.workspace_id = ?)
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

	// Expand each (? IS NULL OR col = ?) pair.
	args := []any{
		baseArgs[0], baseArgs[0],
		baseArgs[1], baseArgs[1],
		baseArgs[2], baseArgs[2],
		baseArgs[3], baseArgs[3],
		baseArgs[4], baseArgs[4],
		baseArgs[5], baseArgs[5],
		baseArgs[6], baseArgs[6],
		baseArgs[7], baseArgs[7],
		baseArgs[8], baseArgs[8],
	}

	return query, args
}

// disbursementFlatRow holds one database result row before pivoting.
type disbursementFlatRow struct {
	RowKey            string
	RowID             string
	ColKey            string
	ColID             string
	TotalDisbursement int64
	TransactionCount  int64
	TotalQuantity     int64
}

// pivotFlatDisbursementRows transforms flat SQL result rows into the proto pivot response.
func pivotFlatDisbursementRows(flat []disbursementFlatRow, req *disbreportpb.DisbursementReportRequest) *disbreportpb.DisbursementReportResponse {
	if len(flat) == 0 {
		return &disbreportpb.DisbursementReportResponse{
			Success: true,
			Summary: buildDisbursementSummary(nil, nil, req),
		}
	}

	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*disbreportpb.DisbursementReportCell)

	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*disbreportpb.DisbursementReportCell
	}
	rowOrder := make([]string, 0)
	rowSeen := make(map[string]bool)
	rowAccums := make(map[string]*rowAccum)

	for _, fr := range flat {
		if !colSeen[fr.ColKey] {
			colSeen[fr.ColKey] = true
			colOrder = append(colOrder, fr.ColKey)
			colID := fr.ColID
			colTotals[fr.ColKey] = &disbreportpb.DisbursementReportCell{
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

		ct := colTotals[fr.ColKey]
		ct.TotalDisbursement += fr.TotalDisbursement
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += float64(fr.TotalQuantity)
	}

	primaryDim := normalizeDisbursementDimension(req.GetPrimaryDimension())
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
	columnTotals := make([]*disbreportpb.DisbursementReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

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
