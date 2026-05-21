package procurementrequest

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// DeleteProcurementRequestRepositories groups repository dependencies.
type DeleteProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// DeleteProcurementRequestServices groups service dependencies.
type DeleteProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteProcurementRequestUseCase handles deleting a procurement request.
type DeleteProcurementRequestUseCase struct {
	repositories DeleteProcurementRequestRepositories
	services     DeleteProcurementRequestServices
}

// NewDeleteProcurementRequestUseCase creates a use case with grouped dependencies.
func NewDeleteProcurementRequestUseCase(
	repositories DeleteProcurementRequestRepositories,
	services DeleteProcurementRequestServices,
) *DeleteProcurementRequestUseCase {
	return &DeleteProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the delete procurement request operation.
func (uc *DeleteProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.DeleteProcurementRequestRequest) (*procurementrequestpb.DeleteProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequest.DeleteProcurementRequest(ctx, req)
}
