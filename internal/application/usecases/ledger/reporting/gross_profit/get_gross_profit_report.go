package gross_profit

import (
	"context"
	"fmt"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// GetGrossProfitReportUseCase handles the business logic for generating a gross profit report.
type GetGrossProfitReportUseCase struct {
	reportingService ports.LedgerReportingService
}

// NewGetGrossProfitReportUseCase creates a new use case with its reporting service dependency.
func NewGetGrossProfitReportUseCase(svc ports.LedgerReportingService) *GetGrossProfitReportUseCase {
	return &GetGrossProfitReportUseCase{reportingService: svc}
}

// Execute validates the request and delegates to the reporting service port.
func (uc *GetGrossProfitReportUseCase) Execute(
	ctx context.Context,
	req *reportpb.GrossProfitReportRequest,
) (*reportpb.GrossProfitReportResponse, error) {
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
