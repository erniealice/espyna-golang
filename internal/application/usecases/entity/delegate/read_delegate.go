package delegate

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
)

// ReadDelegateRepositories groups all repository dependencies
type ReadDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// ReadDelegateServices groups all business service dependencies
type ReadDelegateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadDelegateUseCase(repositories, services)
}

// Execute performs the read delegate operation
func (uc *ReadDelegateUseCase) Execute(ctx context.Context, req *delegatepb.ReadDelegateRequest) (*delegatepb.ReadDelegateResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.authorization_failed", "Authorization failed for Parent/Guardians")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityDelegate, ports.ActionRead)
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
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.data_required", "Parent/Guardian data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.id_required", "Delegate ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Delegate.ReadDelegate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response from repository (includes empty results for not found)

	return resp, nil
}
