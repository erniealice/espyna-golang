package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// GetProcurementRequestItemPageDataRepositories groups repository dependencies.
type GetProcurementRequestItemPageDataRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// GetProcurementRequestItemPageDataServices groups service dependencies.
type GetProcurementRequestItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetProcurementRequestItemPageDataUseCase fetches a single procurement request detail.
type GetProcurementRequestItemPageDataUseCase struct {
	repositories GetProcurementRequestItemPageDataRepositories
	services     GetProcurementRequestItemPageDataServices
}

// NewGetProcurementRequestItemPageDataUseCase creates a use case with grouped dependencies.
func NewGetProcurementRequestItemPageDataUseCase(
	repositories GetProcurementRequestItemPageDataRepositories,
	services GetProcurementRequestItemPageDataServices,
) *GetProcurementRequestItemPageDataUseCase {
	return &GetProcurementRequestItemPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get procurement request item page data operation.
func (uc *GetProcurementRequestItemPageDataUseCase) Execute(ctx context.Context, req *procurementrequestpb.GetProcurementRequestItemPageDataRequest) (*procurementrequestpb.GetProcurementRequestItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.ProcurementRequest.GetProcurementRequestItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load procurement request")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
