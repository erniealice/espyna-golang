//go:build postgres

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
