//go:build postgresql

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

// normalizeCollectionDimension converts camelCase dimension keys to snake_case for SQL switch matching.
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
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
func getCollectionPivotDimensionConfig(tc TableConfig, dimension string) pivotDimensionConfig {
	switch dimension {
	case "monthly":
		expr := "date_trunc('month', tc.payment_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("TO_CHAR(%s, 'Month YYYY')", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "quarterly":
		expr := "date_trunc('quarter', tc.payment_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("'Q' || EXTRACT(QUARTER FROM %s)::int || ' ' || EXTRACT(YEAR FROM %s)::int", expr, expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
		}
	case "yearly":
		expr := "date_trunc('year', tc.payment_date::timestamptz)"
		return pivotDimensionConfig{
			selectKey: fmt.Sprintf("EXTRACT(YEAR FROM %s)::int::text", expr),
			selectID:  fmt.Sprintf("%s::text", expr),
			groupBy:   expr,
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
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries ($1, $2, ...).
//
// The query groups treasury collection records by two independent dimensions:
//   - rowDimension     -> each output row
//   - primaryDimension -> each column within a row (the pivot axis)
func buildCollectionSummaryQuery(tc TableConfig, req *collsumpb.CollectionSummaryRequest, workspaceID string) (string, []any) {
	// Validate and normalise dimensions.
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

	// Combine extra JOINs from both dimensions, deduplicating shared table aliases.
	extraJoins := mergeJoins(rowConfig.extraJoins, colConfig.extraJoins)

	// Build parameter list.
	// $1 = start_date (text or NULL)
	// $2 = end_date   (text or NULL)
	// $3 = client_id (text or NULL)
	// $4 = location_id (text or NULL)
	// $5 = collection_method_id (text or NULL)
	// $6 = currency (text or NULL)
	// $7 = collection_type (text or NULL)
	// $8 = workspace_id (text or NULL)
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
        SUM(tc.amount)::bigint   AS total_collected,
        COUNT(tc.id)::bigint     AS transaction_count
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    %s
    WHERE tc.active = true
      AND ($1::text IS NULL OR tc.payment_date >= $1::date)
      AND ($2::text IS NULL OR tc.payment_date < ($2::date + interval '1 day'))
      AND ($3::text IS NULL OR r.client_id = $3)
      AND ($4::text IS NULL OR r.location_id = $4)
      AND ($5::text IS NULL OR tc.collection_method_id = $5)
      AND ($6::text IS NULL OR tc.currency = $6)
      AND ($7::text IS NULL OR tc.collection_type = $7)
      AND ($8::text IS NULL OR r.workspace_id = $8)
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
// It groups rows by row_key, builds one CollectionSummaryCell per column, and
// computes row totals and report-level summary.
func pivotFlatCollectionRows(flat []collectionFlatRow, req *collsumpb.CollectionSummaryRequest) *collsumpb.CollectionSummaryResponse {
	if len(flat) == 0 {
		return &collsumpb.CollectionSummaryResponse{
			Success: true,
			Summary: buildCollectionSummary(nil, nil, req),
		}
	}

	// Track ordered column keys and column-level totals.
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)
	colTotals := make(map[string]*collsumpb.CollectionSummaryCell)

	// Group flat rows by row_key (preserving insertion order).
	type rowAccum struct {
		rowKey string
		rowID  string
		cells  map[string]*collsumpb.CollectionSummaryCell // colKey -> cell
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
			colTotals[fr.ColKey] = &collsumpb.CollectionSummaryCell{
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

		// Accumulate column totals.
		ct := colTotals[fr.ColKey]
		ct.TotalCollected += fr.TotalCollected
		ct.TransactionCount += fr.TransactionCount
	}

	// Sort columns: periods descending (latest first), entities alphabetical.
	primaryDim := normalizeCollectionDimension(req.GetPrimaryDimension())
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
	columnTotals := make([]*collsumpb.CollectionSummaryCell, 0, len(colOrder))
	for _, ck := range colOrder {
		columnTotals = append(columnTotals, colTotals[ck])
	}

	// Sort rows: periods descending (latest first), entities alphabetical.
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

	// Build rows.
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
				// Emit a zero cell so columns stay aligned.
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
