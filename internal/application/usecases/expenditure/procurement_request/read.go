package procurementrequest

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// ReadProcurementRequestRepositories groups repository dependencies.
type ReadProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// ReadProcurementRequestServices groups service dependencies.
type ReadProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequest.ReadProcurementRequest(ctx, req)
}
