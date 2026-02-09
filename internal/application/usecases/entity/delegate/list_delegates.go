package delegate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// ListDelegatesRepositories groups all repository dependencies
type ListDelegatesRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// ListDelegatesServices groups all business service dependencies
type ListDelegatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListDelegatesUseCase(repositories, services)
}

// Execute performs the list delegates operation
func (uc *ListDelegatesUseCase) Execute(ctx context.Context, req *delegatepb.ListDelegatesRequest) (*delegatepb.ListDelegatesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "Request is required for Parent/Guardians"))
	}

	// Call repository
	resp, err := uc.repositories.Delegate.ListDelegates(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
