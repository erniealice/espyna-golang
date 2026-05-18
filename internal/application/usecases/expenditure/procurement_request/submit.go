package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// SubmitProcurementRequestRepositories groups repository dependencies.
type SubmitProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// SubmitProcurementRequestServices groups service dependencies.
type SubmitProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// SubmitProcurementRequestUseCase transitions a request from DRAFT → SUBMITTED.
type SubmitProcurementRequestUseCase struct {
	repositories SubmitProcurementRequestRepositories
	services     SubmitProcurementRequestServices
}

// NewSubmitProcurementRequestUseCase creates a use case with grouped dependencies.
func NewSubmitProcurementRequestUseCase(
	repositories SubmitProcurementRequestRepositories,
	services SubmitProcurementRequestServices,
) *SubmitProcurementRequestUseCase {
	return &SubmitProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the submit procurement request operation.
func (uc *SubmitProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.SubmitProcurementRequestRequest) (*procurementrequestpb.SubmitProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.ProcurementRequest.SubmitProcurementRequest(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.errors.submit_failed", "[ERR-DEFAULT] Failed to submit procurement request")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
