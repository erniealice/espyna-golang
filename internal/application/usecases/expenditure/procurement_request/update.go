package procurementrequest

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// UpdateProcurementRequestRepositories groups repository dependencies.
type UpdateProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// UpdateProcurementRequestServices groups service dependencies.
type UpdateProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateProcurementRequestUseCase handles updating a procurement request.
type UpdateProcurementRequestUseCase struct {
	repositories UpdateProcurementRequestRepositories
	services     UpdateProcurementRequestServices
}

// NewUpdateProcurementRequestUseCase creates a use case with grouped dependencies.
func NewUpdateProcurementRequestUseCase(
	repositories UpdateProcurementRequestRepositories,
	services UpdateProcurementRequestServices,
) *UpdateProcurementRequestUseCase {
	return &UpdateProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the update procurement request operation.
func (uc *UpdateProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.UpdateProcurementRequestRequest) (*procurementrequestpb.UpdateProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequest.UpdateProcurementRequest(ctx, req)
}
