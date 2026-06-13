package procurementrequest

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// UpdateProcurementRequestRepositories groups repository dependencies.
type UpdateProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// UpdateProcurementRequestServices groups service dependencies.
type UpdateProcurementRequestServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequest.UpdateProcurementRequest(ctx, req)
}
