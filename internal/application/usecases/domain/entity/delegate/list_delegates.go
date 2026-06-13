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

// ListDelegatesRepositories groups all repository dependencies
type ListDelegatesRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// ListDelegatesServices groups all business service dependencies
type ListDelegatesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListDelegatesUseCase handles the business logic for listing delegates
type ListDelegatesUseCase struct {
	repositories ListDelegatesRepositories
	services     ListDelegatesServices
}

// NewListDelegatesUseCase creates use case with grouped dependencies
func NewListDelegatesUseCase(
	repositories ListDelegatesRepositories,
	services ListDelegatesServices,
) *ListDelegatesUseCase {
	return &ListDelegatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListDelegatesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListDelegatesUseCase with grouped parameters instead
func NewListDelegatesUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *ListDelegatesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListDelegatesRepositories{
		Delegate: delegateRepo,
	}

	services := ListDelegatesServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewListDelegatesUseCase(repositories, services)
}

// Execute performs the list delegates operation
func (uc *ListDelegatesUseCase) Execute(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Delegate,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.request_required", "Request is required for Parent/Guardians"))
	}

	// Call repository
	resp, err := uc.repositories.Delegate.ListDelegates(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
