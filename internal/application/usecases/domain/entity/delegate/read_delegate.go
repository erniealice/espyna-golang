package delegate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// ReadDelegateRepositories groups all repository dependencies
type ReadDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// ReadDelegateServices groups all business service dependencies
type ReadDelegateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadDelegateUseCase handles the business logic for reading a delegate
type ReadDelegateUseCase struct {
	repositories ReadDelegateRepositories
	services     ReadDelegateServices
}

// NewReadDelegateUseCase creates use case with grouped dependencies
func NewReadDelegateUseCase(
	repositories ReadDelegateRepositories,
	services ReadDelegateServices,
) *ReadDelegateUseCase {
	return &ReadDelegateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadDelegateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadDelegateUseCase with grouped parameters instead
func NewReadDelegateUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *ReadDelegateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadDelegateRepositories{
		Delegate: delegateRepo,
	}

	services := ReadDelegateServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewReadDelegateUseCase(repositories, services)
}

// Execute performs the read delegate operation
func (uc *ReadDelegateUseCase) Execute(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Delegate,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.request_required", "Request is required for Parent/Guardians"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.data_required", "Parent/Guardian data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.id_required", "Delegate ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Delegate.ReadDelegate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response from repository (includes empty results for not found)

	return resp, nil
}
