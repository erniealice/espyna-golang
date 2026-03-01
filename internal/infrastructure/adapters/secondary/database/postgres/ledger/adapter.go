//go:build postgresql

package ledger

import (
	"context"
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports/domain"
	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// Compile-time interface check
var _ domain.LedgerReportingService = (*LedgerReportingAdapter)(nil)

// TableConfig holds table names for the ledger reporting adapter.
// Unlike entity repositories that use a single table, reporting queries
// span multiple tables, so the adapter needs its own config.
type TableConfig struct {
	Revenue              string
	RevenueLineItem      string
	InventoryTransaction string
	InventoryItem        string
	Product              string
	Location             string
	RevenueCategory      string
	Expenditure          string
}

// LedgerReportingAdapter implements ports.LedgerReportingService using PostgreSQL.
// Unlike entity repositories that use the registry pattern,
// the ledger reporting adapter is instantiated directly because it
// spans multiple tables and doesn't follow the single-entity CRUD model.
//
// Current implementation: queries raw revenue + inventory tables directly.
// Future: when Journal Entries are implemented, this adapter will be
// replaced with one that queries journal entries by account type.
type LedgerReportingAdapter struct {
	db          *sql.DB
	tableConfig TableConfig
}

// NewLedgerReportingAdapter creates a new ledger reporting adapter.
func NewLedgerReportingAdapter(db *sql.DB, config TableConfig) *LedgerReportingAdapter {
	return &LedgerReportingAdapter{db: db, tableConfig: config}
}

// GetGrossProfitReport executes a CTE-based SQL query to compute gross profit
// grouped by the requested dimension (product, location, category, or period).
func (a *LedgerReportingAdapter) GetGrossProfitReport(
	ctx context.Context,
	req *reportpb.GrossProfitReportRequest,
) (*reportpb.GrossProfitReportResponse, error) {
	query, args := buildGrossProfitQuery(a.tableConfig, req)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []*reportpb.GrossProfitLineItem
	for rows.Next() {
		var item reportpb.GrossProfitLineItem
		var groupID sql.NullString
		if err := rows.Scan(
			&item.GroupKey,
			&groupID,
			&item.TotalRevenue,
			&item.TotalDiscount,
			&item.NetRevenue,
			&item.CostOfGoodsSold,
			&item.GrossProfit,
			&item.GrossProfitMargin,
			&item.UnitsSold,
			&item.TransactionCount,
		); err != nil {
			return nil, err
		}
		if groupID.Valid {
			item.GroupId = &groupID.String
		}
		lineItems = append(lineItems, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary := computeSummary(lineItems, req)

	return &reportpb.GrossProfitReportResponse{
		LineItems: lineItems,
		Summary:   summary,
		Success:   true,
	}, nil
}

// ListRevenue returns all revenue records for the report listing.
func (a *LedgerReportingAdapter) ListRevenue(ctx context.Context) ([]map[string]any, error) {
	query := `SELECT id, reference_number, status, total_amount, currency,
		COALESCE(customer_first_name, '') || ' ' || COALESCE(customer_last_name, '') AS customer_name,
		COALESCE(notes, '') AS notes,
		COALESCE(location_name, '') AS location_name,
		created_at
		FROM ` + a.tableConfig.Revenue + `
		ORDER BY created_at DESC
		LIMIT 200`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// ListExpenses returns all expenditure records of type "expense".
func (a *LedgerReportingAdapter) ListExpenses(ctx context.Context) ([]map[string]any, error) {
	table := a.tableConfig.Expenditure
	if table == "" {
		return nil, nil
	}
	query := `SELECT id, reference_number, vendor_name, category, status,
		total_amount, currency,
		COALESCE(expenditure_date_string, '') AS expenditure_date,
		COALESCE(notes, '') AS notes
		FROM ` + table + `
		WHERE expenditure_type = 'expense'
		ORDER BY created_at DESC
		LIMIT 200`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// scanToMaps converts sql.Rows into a slice of column→value maps.
func scanToMaps(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = values[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// computeSummary aggregates line item totals into a report summary.
func computeSummary(items []*reportpb.GrossProfitLineItem, req *reportpb.GrossProfitReportRequest) *reportpb.GrossProfitSummary {
	s := &reportpb.GrossProfitSummary{}
	for _, item := range items {
		s.TotalRevenue += item.TotalRevenue
		s.TotalDiscount += item.TotalDiscount
		s.NetRevenue += item.NetRevenue
		s.TotalCogs += item.CostOfGoodsSold
		s.TotalGrossProfit += item.GrossProfit
		s.TotalUnitsSold += item.UnitsSold
		s.TotalTransactions += item.TransactionCount
	}
	if s.NetRevenue > 0 {
		s.OverallMargin = (s.TotalGrossProfit / s.NetRevenue) * 100
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
	return s
}
