package domain

import (
	"context"
	"time"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// LedgerReportingService defines the contract for ledger reporting operations.
// Unlike entity services which implement gRPC server interfaces,
// reporting services use a custom interface because reports are
// computed aggregates, not stored entities.
//
// Future: This file will also contain AccountService and JournalService
// interfaces when Chart of Accounts and Journal Entries are implemented.
type LedgerReportingService interface {
	GetGrossProfitReport(ctx context.Context, req *reportpb.GrossProfitReportRequest) (*reportpb.GrossProfitReportResponse, error)
	ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
	ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
}
