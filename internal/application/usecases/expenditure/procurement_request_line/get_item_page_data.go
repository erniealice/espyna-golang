package procurementrequestline

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// GetProcurementRequestLineItemPageDataRepositories groups repository dependencies.
type GetProcurementRequestLineItemPageDataRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// GetProcurementRequestLineItemPageDataServices groups service dependencies.
type GetProcurementRequestLineItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetProcurementRequestLineItemPageDataUseCase fetches a single request line detail.
type GetProcurementRequestLineItemPageDataUseCase struct {
	repositories GetProcurementRequestLineItemPageDataRepositories
	services     GetProcurementRequestLineItemPageDataServices
}

// NewGetProcurementRequestLineItemPageDataUseCase creates a use case with grouped dependencies.
func NewGetProcurementRequestLineItemPageDataUseCase(
	repositories GetProcurementRequestLineItemPageDataRepositories,
	services GetProcurementRequestLineItemPageDataServices,
) *GetProcurementRequestLineItemPageDataUseCase {
	return &GetProcurementRequestLineItemPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get procurement request line item page data operation.
func (uc *GetProcurementRequestLineItemPageDataUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.GetProcurementRequestLineItemPageDataRequest) (*procurementrequestlinepb.GetProcurementRequestLineItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequestLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestLineId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request_line.validation.id_required", "Procurement request line ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.ProcurementRequestLine.GetProcurementRequestLineItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request_line.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load procurement request line")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
