package consumer

import (
	"context"
	"database/sql"
	"time"

	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	agingpb       "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	payagingpb    "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"
	clientstmtpb  "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	expreportpb   "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
	reportpb      "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	revreportpb   "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
	collsumpb     "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
	disbreportpb  "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
	suppstmtpb    "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
)

// LedgerReportingService provides access to ledger reporting queries.
// Consumer apps can use this interface or pass the result directly to
// packages that define compatible interfaces (e.g. fycha.DataSource).
type LedgerReportingService interface {
	GetGrossProfitReport(ctx context.Context, req *reportpb.GrossProfitReportRequest) (*reportpb.GrossProfitReportResponse, error)
	GetRevenueReport(ctx context.Context, req *revreportpb.RevenueReportRequest) (*revreportpb.RevenueReportResponse, error)
	GetExpenditureReport(ctx context.Context, req *expreportpb.ExpenditureReportRequest) (*expreportpb.ExpenditureReportResponse, error)
	GetDisbursementReport(ctx context.Context, req *disbreportpb.DisbursementReportRequest) (*disbreportpb.DisbursementReportResponse, error)
	GetReceivablesAgingReport(ctx context.Context, req *agingpb.ReceivablesAgingRequest) (*agingpb.ReceivablesAgingResponse, error)
	GetPayablesAgingReport(ctx context.Context, req *payagingpb.PayablesAgingRequest) (*payagingpb.PayablesAgingResponse, error)
	GetCollectionSummaryReport(ctx context.Context, req *collsumpb.CollectionSummaryRequest) (*collsumpb.CollectionSummaryResponse, error)
	GetClientStatement(ctx context.Context, req *clientstmtpb.ClientStatementRequest) (*clientstmtpb.ClientStatementResponse, error)
	GetSupplierStatement(ctx context.Context, req *suppstmtpb.SupplierStatementRequest) (*suppstmtpb.SupplierStatementResponse, error)
	GetSupplierBalances(ctx context.Context) (map[string]int64, error)
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
	ExpenditureLineItem  string // expenditure_line_item table
	ExpenditureCategory  string // expenditure_category table
	Supplier             string // supplier table
	ProductCollection    string // product_collection table (product <-> product line join) — deprecated, kept for backward compat
	Collection           string // collection table (product line / product_line) — deprecated, kept for backward compat
	Line                 string // line table (product taxonomy — product.line_id FK target)
	LocationArea         string // location_area table
	TreasuryDisbursement string // treasury_disbursement table
	DisbursementMethod   string // disbursement_method table
	SupplierCategory     string // supplier_category table
	Client               string // client table
	ClientCategory       string // client_category table
	Category             string // category table (parent categories)
	TreasuryCollection   string // treasury_collection table
	CollectionMethod     string // collection_method table
	PaymentTerm          string // payment_term table
}

// NewLedgerReportingService creates a new ledger reporting service using registry discovery.
// If no ledger provider has been registered (e.g. via contrib/postgres init()),
// this returns nil — the consumer app should handle nil gracefully (reports will be unavailable).
func NewLedgerReportingService(db *sql.DB, config LedgerReportingTableConfig) LedgerReportingService {
	factory, ok := registry.GetLedgerReportingFactory()
	if !ok {
		return nil // no ledger provider registered (equivalent to old noop)
	}
	result := factory(db, config)
	if svc, ok := result.(LedgerReportingService); ok {
		return svc
	}
	return nil
}
