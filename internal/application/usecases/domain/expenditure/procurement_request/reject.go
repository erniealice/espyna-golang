package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// RejectProcurementRequestRepositories groups repository dependencies.
type RejectProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// RejectProcurementRequestServices groups service dependencies.
type RejectProcurementRequestServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// RejectProcurementRequestUseCase transitions a request from PENDING_APPROVAL → REJECTED.
// A rejection_reason is strongly recommended but not required at the use case level.
type RejectProcurementRequestUseCase struct {
	repositories RejectProcurementRequestRepositories
	services     RejectProcurementRequestServices
}

// NewRejectProcurementRequestUseCase creates a use case with grouped dependencies.
func NewRejectProcurementRequestUseCase(
	repositories RejectProcurementRequestRepositories,
	services RejectProcurementRequestServices,
) *RejectProcurementRequestUseCase {
	return &RejectProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the reject procurement request operation.
func (uc *RejectProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.RejectProcurementRequestRequest) (*procurementrequestpb.RejectProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityProcurementRequest, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.ProcurementRequest.RejectProcurementRequest(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.errors.reject_failed", "[ERR-DEFAULT] Failed to reject procurement request")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
