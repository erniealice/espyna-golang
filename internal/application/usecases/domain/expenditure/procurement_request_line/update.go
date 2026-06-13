package procurementrequestline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// UpdateProcurementRequestLineRepositories groups repository dependencies.
type UpdateProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// UpdateProcurementRequestLineServices groups service dependencies.
type UpdateProcurementRequestLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateProcurementRequestLineUseCase handles updating a procurement request line.
type UpdateProcurementRequestLineUseCase struct {
	repositories UpdateProcurementRequestLineRepositories
	services     UpdateProcurementRequestLineServices
}

// NewUpdateProcurementRequestLineUseCase creates a use case with grouped dependencies.
func NewUpdateProcurementRequestLineUseCase(
	repositories UpdateProcurementRequestLineRepositories,
	services UpdateProcurementRequestLineServices,
) *UpdateProcurementRequestLineUseCase {
	return &UpdateProcurementRequestLineUseCase{repositories: repositories, services: services}
}

// Execute performs the update procurement request line operation.
func (uc *UpdateProcurementRequestLineUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.UpdateProcurementRequestLineRequest) (*procurementrequestlinepb.UpdateProcurementRequestLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequestLine,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.validation.id_required", "Procurement request line ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequestLine.UpdateProcurementRequestLine(ctx, req)
}
