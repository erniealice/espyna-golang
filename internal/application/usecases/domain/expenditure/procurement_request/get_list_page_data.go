package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// GetProcurementRequestListPageDataRepositories groups repository dependencies.
type GetProcurementRequestListPageDataRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// GetProcurementRequestListPageDataServices groups service dependencies.
type GetProcurementRequestListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetProcurementRequestListPageDataUseCase fetches paginated procurement request list data.
type GetProcurementRequestListPageDataUseCase struct {
	repositories GetProcurementRequestListPageDataRepositories
	services     GetProcurementRequestListPageDataServices
}

// NewGetProcurementRequestListPageDataUseCase creates a use case with grouped dependencies.
func NewGetProcurementRequestListPageDataUseCase(
	repositories GetProcurementRequestListPageDataRepositories,
	services GetProcurementRequestListPageDataServices,
) *GetProcurementRequestListPageDataUseCase {
	return &GetProcurementRequestListPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get procurement request list page data operation.
func (uc *GetProcurementRequestListPageDataUseCase) Execute(ctx context.Context, req *procurementrequestpb.GetProcurementRequestListPageDataRequest) (*procurementrequestpb.GetProcurementRequestListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	resp, err := uc.repositories.ProcurementRequest.GetProcurementRequestListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load procurement request list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
