package simple_payables_aging

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// GetSimplePayablesAgingReportUseCase handles the business logic for generating a
// simplified payables aging report grouped by supplier name.
type GetSimplePayablesAgingReportUseCase struct {
	reportingService     ports.LedgerReportingService
	authorizationService ports.AuthorizationService
	translationService   ports.TranslationService
}

// NewGetSimplePayablesAgingReportUseCase creates a new use case with its reporting service dependency.
func NewGetSimplePayablesAgingReportUseCase(svc ports.LedgerReportingService) *GetSimplePayablesAgingReportUseCase {
	return &GetSimplePayablesAgingReportUseCase{
		reportingService:     svc,
		authorizationService: nil,
		translationService:   ports.NewNoOpTranslationService(),
	}
}

// Execute validates the request and delegates to the reporting service port.
func (uc *GetSimplePayablesAgingReportUseCase) Execute(
	ctx context.Context,
	req *reportpb.PayablesAgingReportRequest,
) (*reportpb.PayablesAgingReportResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.authorizationService, uc.translationService,
		"reports", "view"); err != nil {
		return nil, err
	}

	// Delegate to port — SQL aggregation lives in the adapter layer
	return uc.reportingService.GetSimplePayablesAgingReport(ctx, req)
}
