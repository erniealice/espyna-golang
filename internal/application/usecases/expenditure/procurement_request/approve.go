package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// ApproveProcurementRequestRepositories groups repository dependencies.
type ApproveProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// ApproveProcurementRequestServices groups service dependencies.
type ApproveProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ApproveProcurementRequestUseCase transitions a request from PENDING_APPROVAL → APPROVED.
type ApproveProcurementRequestUseCase struct {
	repositories ApproveProcurementRequestRepositories
	services     ApproveProcurementRequestServices
}

// NewApproveProcurementRequestUseCase creates a use case with grouped dependencies.
func NewApproveProcurementRequestUseCase(
	repositories ApproveProcurementRequestRepositories,
	services ApproveProcurementRequestServices,
) *ApproveProcurementRequestUseCase {
	return &ApproveProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the approve procurement request operation.
func (uc *ApproveProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.ApproveProcurementRequestRequest) (*procurementrequestpb.ApproveProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	if req.ApprovedBy == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.approved_by_required", "Approver ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.ProcurementRequest.ApproveProcurementRequest(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.errors.approve_failed", "[ERR-DEFAULT] Failed to approve procurement request")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
