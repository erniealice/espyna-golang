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

// DeleteProcurementRequestLineRepositories groups repository dependencies.
type DeleteProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// DeleteProcurementRequestLineServices groups service dependencies.
type DeleteProcurementRequestLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequestLine,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.validation.id_required", "Procurement request line ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequestLine.DeleteProcurementRequestLine(ctx, req)
}
