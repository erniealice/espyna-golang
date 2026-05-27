//go:build sqlserver

package ledger

import (
	"fmt"
	"sort"
	"strings"

	expreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
)

// validExpenditurePivotDimensions is a whitelist of allowed dimension values for expenditure reports.
var validExpenditurePivotDimensions = map[string]bool{
	"monthly":           true,
	"quarterly":         true,
	"yearly":            true,
	"product":           true,
	"product_line":      true,
	"productLine":       true,
	"location":          true,
	"location_area":     true,
	"locationArea":      true,
	"category":          true,
	"supplier":          true,
	"expenditure_type":  true,
	"expenditureType":   true,
	"supplierCategory":  true,
	"supplier_category": true,
}

// normalizeExpenditureDimension converts camelCase dimension keys to snake_case.
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
// SQL Server period expressions use DATEFROMPARTS / YEAR / MONTH instead of date_trunc / TO_CHAR.
func getExpenditurePivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		expr := "DATEFROMPARTS(YEAR(e.expenditure_date), MONTH(e.expenditure_date), 1)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("FORMAT(%s, 'MMMM yyyy')", expr),
			selectID:  fmt.Sprintf("CAST(%s AS nvarchar(30))", expr),
			groupBy:   "YEAR(e.expenditure_date), MONTH(e.expenditure_date)",
		}
	case "quarterly":
		return pivotDimensionConfig{
			selectKey: "'Q' + CAST(DATEPART(quarter, e.expenditure_date) AS nvarchar) + ' ' + CAST(YEAR(e.expenditure_date) AS nvarchar)",
			selectID:  "CAST(DATEFROMPARTS(YEAR(e.expenditure_date), ((MONTH(e.expenditure_date) - 1) / 3) * 3 + 1, 1) AS nvarchar(30))",
			groupBy:   "YEAR(e.expenditure_date), DATEPART(quarter, e.expenditure_date)",
		}
	case "yearly":
		return pivotDimensionConfig{
			selectKey: "CAST(YEAR(e.expenditure_date) AS nvarchar(10))",
			selectID:  "CAST(DATEFROMPARTS(YEAR(e.expenditure_date), 1, 1) AS nvarchar(30))",
			groupBy:   "YEAR(e.expenditure_date)",
		}
	case "product":
		return pivotDimensionConfig{
			selectKey: "COALESCE(p.name, 'Unassigned')",
			selectID:  "COALESCE(eli.product_id, '__none__')",
			groupBy:   "eli.product_id, p.name",
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
			selectKey:  "COALESCE(s.name, 'No Supplier')",
			selectID:   "COALESCE(e.supplier_id, '__none__')",
			groupBy:    "e.supplier_id, s.name",
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
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - ::timestamptz IS NULL → @pN IS NULL.
//   - date_trunc / TO_CHAR → DATEFROMPARTS / FORMAT / YEAR / MONTH.
//   - ::bigint → CAST(… AS bigint).
func buildExpenditureReportQuery(tc TableConfig, req *expreportpb.ExpenditureReportRequest, workspaceID string) (string, []any) {
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

	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// @p1 = start_date (datetime2 or NULL)
	// @p2 = end_date   (datetime2 or NULL)
	// @p3 = product_id (text or NULL)
	// @p4 = location_id (text or NULL)
	// @p5 = expenditure_category_id (text or NULL)
	// @p6 = supplier_id (text or NULL)
	// @p7 = expenditure_type (text or NULL)
	// @p8 = location_area_id (text or NULL)
	// @p9 = workspace_id (text or NULL)
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
        CAST(SUM(eli.line_amount) AS bigint) AS total_expenditure,
        COUNT(DISTINCT e.id)                 AS transaction_count,
        CAST(SUM(eli.quantity) AS bigint)    AS total_quantity
    FROM %s eli
    JOIN %s e ON e.id = eli.expenditure_id
    LEFT JOIN %s p ON p.id = eli.product_id
    LEFT JOIN %s l ON l.id = e.location_id
    %s
    WHERE e.active = 1
      AND e.status NOT IN ('cancelled', 'draft')
      AND (@p1 IS NULL OR e.expenditure_date >= @p1)
      AND (@p2 IS NULL OR e.expenditure_date <= @p2)
      AND (@p3 IS NULL OR eli.product_id = @p3)
      AND (@p4 IS NULL OR e.location_id = @p4)
      AND (@p5 IS NULL OR e.expenditure_category_id = @p5)
      AND (@p6 IS NULL OR e.supplier_id = @p6)
      AND (@p7 IS NULL OR e.expenditure_type = @p7)
      AND (@p8 IS NULL OR l.location_area_id = @p8)
      AND (@p9 IS NULL OR e.workspace_id = @p9)
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
	RowKey           string
	RowID            string
	ColKey           string
	ColID            string
	TotalExpenditure int64
	TransactionCount int64
	TotalQuantity    float64
}

// pivotFlatExpenditureRows transforms flat SQL result rows into the proto pivot response.
func pivotFlatExpenditureRows(flat []expenditureFlatRow, req *expreportpb.ExpenditureReportRequest) *expreportpb.ExpenditureReportResponse {
	if len(flat) == 0 {
		return &expreportpb.ExpenditureReportResponse{
			Success: true,
			Summary: buildExpenditureSummary(nil, nil, req),
		}
	}

	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*expreportpb.ExpenditureReportCell)

	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*expreportpb.ExpenditureReportCell
	}
	rowOrder := make([]string, 0)
	rowSeen := make(map[string]bool)
	rowAccums := make(map[string]*rowAccum)

	for _, fr := range flat {
		if !colSeen[fr.ColKey] {
			colSeen[fr.ColKey] = true
			colOrder = append(colOrder, fr.ColKey)
			colID := fr.ColID
			colTotals[fr.ColKey] = &expreportpb.ExpenditureReportCell{
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

		ct := colTotals[fr.ColKey]
		ct.TotalExpenditure += fr.TotalExpenditure
		ct.TransactionCount += fr.TransactionCount
		ct.TotalQuantity += fr.TotalQuantity
	}

	primaryDim := normalizeExpenditureDimension(req.GetPrimaryDimension())
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

	columnTotals := make([]*expreportpb.ExpenditureReportCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

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
