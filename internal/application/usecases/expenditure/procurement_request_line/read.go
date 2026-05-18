package procurementrequestline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// ReadProcurementRequestLineRepositories groups repository dependencies.
type ReadProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// ReadProcurementRequestLineServices groups service dependencies.
type ReadProcurementRequestLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadProcurementRequestLineUseCase handles reading a procurement request line.
type ReadProcurementRequestLineUseCase struct {
	repositories ReadProcurementRequestLineRepositories
	services     ReadProcurementRequestLineServices
}

// NewReadProcurementRequestLineUseCase creates a use case with grouped dependencies.
func NewReadProcurementRequestLineUseCase(
	repositories ReadProcurementRequestLineRepositories,
	services ReadProcurementRequestLineServices,
) *ReadProcurementRequestLineUseCase {
	return &ReadProcurementRequestLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read procurement request line operation.
func (uc *ReadProcurementRequestLineUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.ReadProcurementRequestLineRequest) (*procurementrequestlinepb.ReadProcurementRequestLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequestLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request_line.validation.id_required", "Procurement request line ID is required [DEFAULT]"))
	}
	return uc.repositories.ProcurementRequestLine.ReadProcurementRequestLine(ctx, req)
}
