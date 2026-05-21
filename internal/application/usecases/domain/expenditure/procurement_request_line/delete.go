package procurementrequestline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// DeleteProcurementRequestLineRepositories groups repository dependencies.
type DeleteProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// DeleteProcurementRequestLineServices groups service dependencies.
type DeleteProcurementRequestLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteProcurementRequestLineUseCase handles deleting a procurement request line.
type DeleteProcurementRequestLineUseCase struct {
	repositories DeleteProcurementRequestLineRepositories
	services     DeleteProcurementRequestLineServices
}

// NewDeleteProcurementRequestLineUseCase creates a use case with grouped dependencies.
func NewDeleteProcurementRequestLineUseCase(
	repositories DeleteProcurementRequestLineRepositories,
	services DeleteProcurementRequestLineServices,
) *DeleteProcurementRequestLineUseCase {
	return &DeleteProcurementRequestLineUseCase{repositories: repositories, services: services}
}

// Execute performs the delete procurement request line operation.
func (uc *DeleteProcurementRequestLineUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.DeleteProcurementRequestLineRequest) (*procurementrequestlinepb.DeleteProcurementRequestLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequestLine, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request_line.validation.id_required", "Procurement request line ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequestLine.DeleteProcurementRequestLine(ctx, req)
}
