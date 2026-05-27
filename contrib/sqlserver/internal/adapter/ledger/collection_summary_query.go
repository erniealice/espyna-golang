//go:build sqlserver

package ledger

import (
	"fmt"
	"sort"
	"strings"

	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
)

// validCollectionDimensions is a whitelist of allowed dimension values for collection summary reports.
var validCollectionDimensions = map[string]bool{
	"monthly":           true,
	"quarterly":         true,
	"yearly":            true,
	"client":            true,
	"client_category":   true,
	"clientCategory":    true,
	"location":          true,
	"location_area":     true,
	"locationArea":      true,
	"collection_method": true,
	"collectionMethod":  true,
	"collection_type":   true,
	"collectionType":    true,
}

// normalizeCollectionDimension converts camelCase dimension keys to snake_case.
func normalizeCollectionDimension(dim string) string {
	switch dim {
	case "clientCategory":
		return "client_category"
	case "locationArea":
		return "location_area"
	case "collectionMethod":
		return "collection_method"
	case "collectionType":
		return "collection_type"
	default:
		return dim
	}
}

// getCollectionPivotDimensionConfig returns SQL fragments for the requested collection dimension.
// SQL Server differences: DATEFROMPARTS / FORMAT instead of date_trunc / TO_CHAR.
// payment_date in treasury_collection is a TEXT column (YYYY-MM-DD); cast with CAST(… AS date).
func getCollectionPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		// payment_date is TEXT 'YYYY-MM-DD'; truncate to month start.
		expr := "DATEFROMPARTS(YEAR(CAST(tc.payment_date AS date)), MONTH(CAST(tc.payment_date AS date)), 1)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("FORMAT(%s, 'MMMM yyyy')", expr),
			selectID:  fmt.Sprintf("CAST(%s AS nvarchar(30))", expr),
			groupBy:   "YEAR(CAST(tc.payment_date AS date)), MONTH(CAST(tc.payment_date AS date))",
		}
	case "quarterly":
		return pivotDimensionConfig{
			selectKey: "'Q' + CAST(DATEPART(quarter, CAST(tc.payment_date AS date)) AS nvarchar) + ' ' + CAST(YEAR(CAST(tc.payment_date AS date)) AS nvarchar)",
			selectID:  "CAST(DATEFROMPARTS(YEAR(CAST(tc.payment_date AS date)), ((MONTH(CAST(tc.payment_date AS date)) - 1) / 3) * 3 + 1, 1) AS nvarchar(30))",
			groupBy:   "YEAR(CAST(tc.payment_date AS date)), DATEPART(quarter, CAST(tc.payment_date AS date))",
		}
	case "yearly":
		return pivotDimensionConfig{
			selectKey: "CAST(YEAR(CAST(tc.payment_date AS date)) AS nvarchar(10))",
			selectID:  "CAST(DATEFROMPARTS(YEAR(CAST(tc.payment_date AS date)), 1, 1) AS nvarchar(30))",
			groupBy:   "YEAR(CAST(tc.payment_date AS date))",
		}
	case "client":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(cl.name, r.name, 'Unassigned')",
			selectID:   "COALESCE(r.client_id, '__none__')",
			groupBy:    "r.client_id, cl.name, r.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s cl ON cl.id = r.client_id", tc.Client),
		}
	case "client_category":
		return pivotDimensionConfig{
			selectKey: "COALESCE(cat.name, 'Unassigned')",
			selectID:  "COALESCE(cc.category_id, '__none__')",
			groupBy:   "cc.category_id, cat.name",
			extraJoins: fmt.Sprintf(
				"LEFT JOIN %s cl ON cl.id = r.client_id LEFT JOIN %s cc ON cc.id = cl.category_id LEFT JOIN %s cat ON cat.id = cc.category_id",
				tc.Client, tc.ClientCategory, tc.Category),
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
	case "collection_method":
		return pivotDimensionConfig{
			selectKey:  "COALESCE(cm.name, 'Unassigned')",
			selectID:   "COALESCE(tc.collection_method_id, '__none__')",
			groupBy:    "tc.collection_method_id, cm.name",
			extraJoins: fmt.Sprintf("LEFT JOIN %s cm ON cm.id = tc.collection_method_id", tc.CollectionMethod),
		}
	case "collection_type":
		return pivotDimensionConfig{
			selectKey: "COALESCE(tc.collection_type, 'Unassigned')",
			selectID:  "COALESCE(tc.collection_type, '__none__')",
			groupBy:   "tc.collection_type",
		}
	default:
		return getCollectionPivotDimensionConfig(tc, "client")
	}
}

// buildCollectionSummaryQuery constructs the pivot SQL query and its parameter args.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - payment_date text comparisons: plain string comparison (YYYY-MM-DD sorts lexicographically).
//   - interval '1 day' → CONVERT(varchar, DATEADD(day, 1, CAST(@pN AS date)), 23).
//   - ::bigint → CAST(… AS bigint).
func buildCollectionSummaryQuery(tc TableConfig, req *collsumpb.CollectionSummaryRequest, workspaceID string) (string, []any) {
	primaryDim := normalizeCollectionDimension(req.GetPrimaryDimension())
	if !validCollectionDimensions[primaryDim] {
		primaryDim = "monthly"
	}
	rowDim := normalizeCollectionDimension(req.GetRowDimension())
	if !validCollectionDimensions[rowDim] {
		rowDim = "client"
	}

	colConfig := getCollectionPivotDimensionConfig(tc, primaryDim)
	rowConfig := getCollectionPivotDimensionConfig(tc, rowDim)

	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// @p1 = start_date (text or NULL)
	// @p2 = end_date   (text or NULL)
	// @p3 = client_id (text or NULL)
	// @p4 = location_id (text or NULL)
	// @p5 = collection_method_id (text or NULL)
	// @p6 = currency (text or NULL)
	// @p7 = collection_type (text or NULL)
	// @p8 = workspace_id (text or NULL)
	args := []any{
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetClientId()),
		nilIfEmpty(req.GetLocationId()),
		nilIfEmpty(req.GetCollectionMethodId()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(req.GetCollectionType()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH collection_pivot AS (
    SELECT
        %s AS row_key,
        %s AS row_id,
        %s AS col_key,
        %s AS col_id,
        CAST(SUM(tc.amount) AS bigint) AS total_collected,
        CAST(COUNT(tc.id) AS bigint)   AS transaction_count
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    %s
    WHERE tc.active = 1
      AND (@p1 IS NULL OR tc.payment_date >= @p1)
      AND (@p2 IS NULL OR tc.payment_date < CONVERT(varchar, DATEADD(day, 1, CAST(@p2 AS date)), 23))
      AND (@p3 IS NULL OR r.client_id = @p3)
      AND (@p4 IS NULL OR r.location_id = @p4)
      AND (@p5 IS NULL OR tc.collection_method_id = @p5)
      AND (@p6 IS NULL OR tc.currency = @p6)
      AND (@p7 IS NULL OR tc.collection_type = @p7)
      AND (@p8 IS NULL OR r.workspace_id = @p8)
    GROUP BY %s, %s
)
SELECT row_key, row_id, col_key, col_id,
       total_collected, transaction_count
FROM collection_pivot
ORDER BY row_key, col_key`,
		rowConfig.selectKey, rowConfig.selectID,
		colConfig.selectKey, colConfig.selectID,
		tc.TreasuryCollection,
		tc.Revenue,
		extraJoins,
		rowConfig.groupBy, colConfig.groupBy,
	)

	return query, args
}

// collectionFlatRow holds one database result row before pivoting.
type collectionFlatRow struct {
	RowKey           string
	RowID            string
	ColKey           string
	ColID            string
	TotalCollected   int64
	TransactionCount int64
}

// pivotFlatCollectionRows transforms flat SQL result rows into the proto pivot response.
func pivotFlatCollectionRows(flat []collectionFlatRow, req *collsumpb.CollectionSummaryRequest) *collsumpb.CollectionSummaryResponse {
	if len(flat) == 0 {
		return &collsumpb.CollectionSummaryResponse{
			Success: true,
			Summary: buildCollectionSummary(nil, nil, req),
		}
	}

	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*collsumpb.CollectionSummaryCell)

	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*collsumpb.CollectionSummaryCell
	}
	rowOrder := make([]string, 0)
	rowSeen := make(map[string]bool)
	rowAccums := make(map[string]*rowAccum)

	for _, fr := range flat {
		if !colSeen[fr.ColKey] {
			colSeen[fr.ColKey] = true
			colOrder = append(colOrder, fr.ColKey)
			colID := fr.ColID
			colTotals[fr.ColKey] = &collsumpb.CollectionSummaryCell{
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
				cells:  make(map[string]*collsumpb.CollectionSummaryCell),
			}
		}

		ra := rowAccums[fr.RowKey]
		if _, ok := ra.cells[fr.ColKey]; !ok {
			colID := fr.ColID
			ra.cells[fr.ColKey] = &collsumpb.CollectionSummaryCell{
				ColumnKey: fr.ColKey,
				ColumnId:  &colID,
			}
		}
		cell := ra.cells[fr.ColKey]
		cell.TotalCollected += fr.TotalCollected
		cell.TransactionCount += fr.TransactionCount

		ct := colTotals[fr.ColKey]
		ct.TotalCollected += fr.TotalCollected
		ct.TransactionCount += fr.TransactionCount
	}

	primaryDim := normalizeCollectionDimension(req.GetPrimaryDimension())
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

	columnTotals := make([]*collsumpb.CollectionSummaryCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	rowDimNorm := normalizeCollectionDimension(req.GetRowDimension())
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

	pbRows := make([]*collsumpb.CollectionSummaryRow, 0, len(rowOrder))
	for _, rk := range rowOrder {
		ra := rowAccums[rk]
		cells := make([]*collsumpb.CollectionSummaryCell, 0, len(colOrder))
		var rowTotal int64
		var rowTxCount int64
		for _, ck := range colOrder {
			if cell, ok := ra.cells[ck]; ok {
				cells = append(cells, cell)
				rowTotal += cell.TotalCollected
				rowTxCount += cell.TransactionCount
			} else {
				colID := colTotals[ck].GetColumnId()
				cells = append(cells, &collsumpb.CollectionSummaryCell{
					ColumnKey: ck,
					ColumnId:  &colID,
				})
			}
		}
		rowID := ra.rowID
		pbRows = append(pbRows, &collsumpb.CollectionSummaryRow{
			RowKey:              ra.rowKey,
			RowId:               &rowID,
			Cells:               cells,
			RowTotal:            rowTotal,
			RowTransactionCount: rowTxCount,
		})
	}

	summary := buildCollectionSummary(pbRows, columnTotals, req)

	return &collsumpb.CollectionSummaryResponse{
		ColumnKeys: colHeaders,
		Rows:       pbRows,
		Summary:    summary,
		Success:    true,
	}
}

// buildCollectionSummary computes report-level totals from assembled rows.
func buildCollectionSummary(rows []*collsumpb.CollectionSummaryRow, columnTotals []*collsumpb.CollectionSummaryCell, req *collsumpb.CollectionSummaryRequest) *collsumpb.CollectionSummarySummary {
	s := &collsumpb.CollectionSummarySummary{
		ColumnTotals: columnTotals,
	}
	for _, row := range rows {
		s.GrandTotal += row.RowTotal
		s.TotalTransactions += row.RowTransactionCount
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
