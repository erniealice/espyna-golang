package cash_book

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// GetCashBookReportUseCase handles the business logic for generating a cash book report.
type GetCashBookReportUseCase struct {
	reportingService     ports.LedgerReportingService
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetCashBookReportUseCase creates a new use case with its reporting service dependency.
func NewGetCashBookReportUseCase(svc ports.LedgerReportingService) *GetCashBookReportUseCase {
	return &GetCashBookReportUseCase{
		reportingService:     svc,
		authorizationService: nil,
		translationService:   ports.NewNoOpTranslator(),
	}
}

// Execute validates the request and delegates to the reporting service port.
func (uc *GetCashBookReportUseCase) Execute(
	ctx context.Context,
	req *reportpb.CashBookReportRequest,
) (*reportpb.CashBookReportResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.authorizationService, uc.translationService,
		"reports", "view"); err != nil {
		return nil, err
	}

	// Delegate to port — SQL aggregation lives in the adapter layer
	return uc.reportingService.GetCashBookReport(ctx, req)
}
