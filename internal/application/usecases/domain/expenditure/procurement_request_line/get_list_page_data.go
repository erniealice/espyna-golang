package procurementrequestline

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// GetProcurementRequestLineListPageDataRepositories groups repository dependencies.
type GetProcurementRequestLineListPageDataRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// GetProcurementRequestLineListPageDataServices groups service dependencies.
type GetProcurementRequestLineListPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// GetProcurementRequestLineListPageDataUseCase fetches paginated request line list data.
type GetProcurementRequestLineListPageDataUseCase struct {
	repositories GetProcurementRequestLineListPageDataRepositories
	services     GetProcurementRequestLineListPageDataServices
}

// NewGetProcurementRequestLineListPageDataUseCase creates a use case with grouped dependencies.
func NewGetProcurementRequestLineListPageDataUseCase(
	repositories GetProcurementRequestLineListPageDataRepositories,
	services GetProcurementRequestLineListPageDataServices,
) *GetProcurementRequestLineListPageDataUseCase {
	return &GetProcurementRequestLineListPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get procurement request line list page data operation.
func (uc *GetProcurementRequestLineListPageDataUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.GetProcurementRequestLineListPageDataRequest) (*procurementrequestlinepb.GetProcurementRequestLineListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityProcurementRequestLine, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	resp, err := uc.repositories.ProcurementRequestLine.GetProcurementRequestLineListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load procurement request line list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
