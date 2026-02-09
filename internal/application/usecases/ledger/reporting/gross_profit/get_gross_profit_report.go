package gross_profit

import (
	"context"
	"fmt"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
)

// GetGrossProfitReportUseCase handles the business logic for generating a gross profit report.
type GetGrossProfitReportUseCase struct {
	reportingService     ports.LedgerReportingService
	authorizationService ports.AuthorizationService
	translationService   ports.TranslationService
}

// NewGetGrossProfitReportUseCase creates a new use case with its reporting service dependency.
func NewGetGrossProfitReportUseCase(svc ports.LedgerReportingService) *GetGrossProfitReportUseCase {
	return &GetGrossProfitReportUseCase{
		reportingService:     svc,
		authorizationService: nil,
		translationService:   ports.NewNoOpTranslationService(),
	}
}

// NewGetGrossProfitReportUseCaseWithServices creates a new use case with full service dependencies.
func NewGetGrossProfitReportUseCaseWithServices(
	svc ports.LedgerReportingService,
	authService ports.AuthorizationService,
	translationService ports.TranslationService,
) *GetGrossProfitReportUseCase {
	return &GetGrossProfitReportUseCase{
		reportingService:     svc,
		authorizationService: authService,
		translationService:   translationService,
	}
}

// Execute validates the request and delegates to the reporting service port.
func (uc *GetGrossProfitReportUseCase) Execute(
	ctx context.Context,
	req *reportpb.GrossProfitReportRequest,
) (*reportpb.GrossProfitReportResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.authorizationService, uc.translationService,
		"reports", "view"); err != nil {
		return nil, err
	}

	// Validate group_by if provided
	if req.GroupBy != nil {
		validGroups := map[string]bool{"product": true, "location": true, "category": true, "period": true}
		if !validGroups[*req.GroupBy] {
			return nil, fmt.Errorf("invalid group_by value: %s", *req.GroupBy)
		}
	}

	// Delegate to port — SQL aggregation lives in the adapter layer
	return uc.reportingService.GetGrossProfitReport(ctx, req)
}
