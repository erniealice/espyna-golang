package delegate

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
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
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.authorization_failed", "Authorization failed for Parent/Guardians")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityDelegate, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.authorization_failed", "Authorization failed for Parent/Guardians")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.authorization_failed", "Authorization failed for Parent/Guardians")
		return nil, errors.New(translatedError)
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
