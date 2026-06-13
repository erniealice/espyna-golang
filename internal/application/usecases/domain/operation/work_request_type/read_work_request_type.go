package work_request_type

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// ReadWorkRequestTypeRepositories groups all repository dependencies
type ReadWorkRequestTypeRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// ReadWorkRequestTypeServices groups all business service dependencies
type ReadWorkRequestTypeServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadWorkRequestTypeUseCase handles the business logic for reading work request types
type ReadWorkRequestTypeUseCase struct {
	repositories ReadWorkRequestTypeRepositories
	services     ReadWorkRequestTypeServices
}

// NewReadWorkRequestTypeUseCase creates use case with grouped dependencies
func NewReadWorkRequestTypeUseCase(
	repositories ReadWorkRequestTypeRepositories,
	services ReadWorkRequestTypeServices,
) *ReadWorkRequestTypeUseCase {
	return &ReadWorkRequestTypeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read work request type operation
func (uc *ReadWorkRequestTypeUseCase) Execute(ctx context.Context, req *work_request_typepb.ReadWorkRequestTypeRequest) (*work_request_typepb.ReadWorkRequestTypeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequestType,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.WorkRequestType.ReadWorkRequestType(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadWorkRequestTypeUseCase) validateInput(ctx context.Context, req *work_request_typepb.ReadWorkRequestTypeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.id_required", "[ERR-DEFAULT] Work request type ID is required"))
	}
	return nil
}
