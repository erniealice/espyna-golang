package domain

import (
	"context"
	"time"

	agingpb        "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	clientstmtpb   "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	expreportpb    "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
	reportpb       "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	revreportpb    "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
	suppstmtpb     "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
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
	GetRevenueReport(ctx context.Context, req *revreportpb.RevenueReportRequest) (*revreportpb.RevenueReportResponse, error)
	GetExpenditureReport(ctx context.Context, req *expreportpb.ExpenditureReportRequest) (*expreportpb.ExpenditureReportResponse, error)
	GetReceivablesAgingReport(ctx context.Context, req *agingpb.ReceivablesAgingRequest) (*agingpb.ReceivablesAgingResponse, error)
	GetClientStatement(ctx context.Context, req *clientstmtpb.ClientStatementRequest) (*clientstmtpb.ClientStatementResponse, error)
	GetSupplierStatement(ctx context.Context, req *suppstmtpb.SupplierStatementRequest) (*suppstmtpb.SupplierStatementResponse, error)
	ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
	ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error)
}
