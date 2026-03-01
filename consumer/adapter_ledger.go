package consumer

import (
	"context"
	"time"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// LedgerReportingService provides access to ledger reporting queries.
// Consumer apps can use this interface or pass the result directly to
// packages that define compatible interfaces (e.g. fycha.DataSource).
type LedgerReportingService interface {
	GetGrossProfitReport(ctx context.Context, req *reportpb.GrossProfitReportRequest) (*reportpb.GrossProfitReportResponse, error)
	ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
	ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
}

// LedgerReportingTableConfig configures table names for ledger reporting queries.
type LedgerReportingTableConfig struct {
	Revenue              string
	RevenueLineItem      string
	InventoryTransaction string
	InventoryItem        string
	Product              string
	Location             string
	RevenueCategory      string
	Expenditure          string
}
