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

// ReadProcurementRequestRepositories groups repository dependencies.
type ReadProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// ReadProcurementRequestServices groups service dependencies.
type ReadProcurementRequestServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadProcurementRequestUseCase handles reading a procurement request.
type ReadProcurementRequestUseCase struct {
	repositories ReadProcurementRequestRepositories
	services     ReadProcurementRequestServices
}

// NewReadProcurementRequestUseCase creates a use case with grouped dependencies.
func NewReadProcurementRequestUseCase(
	repositories ReadProcurementRequestRepositories,
	services ReadProcurementRequestServices,
) *ReadProcurementRequestUseCase {
	return &ReadProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the read procurement request operation.
func (uc *ReadProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.ReadProcurementRequestRequest) (*procurementrequestpb.ReadProcurementRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequest,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequest.ReadProcurementRequest(ctx, req)
}
