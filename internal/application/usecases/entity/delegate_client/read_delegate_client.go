package delegate_client

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

// ReadDelegateClientRepositories groups all repository dependencies
type ReadDelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// ReadDelegateClientServices groups all business service dependencies
type ReadDelegateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadDelegateClientUseCase handles the business logic for reading a delegate client
type ReadDelegateClientUseCase struct {
	repositories ReadDelegateClientRepositories
	services     ReadDelegateClientServices
}

// NewReadDelegateClientUseCase creates use case with grouped dependencies
func NewReadDelegateClientUseCase(
	repositories ReadDelegateClientRepositories,
	services ReadDelegateClientServices,
) *ReadDelegateClientUseCase {
	return &ReadDelegateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadDelegateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadDelegateClientUseCase with grouped parameters instead
func NewReadDelegateClientUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
) *ReadDelegateClientUseCase {
	repositories := ReadDelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       nil,
		Client:         nil,
	}
	services := ReadDelegateClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadDelegateClientUseCase(repositories, services)
}

// Execute performs the read delegate client operation
func (uc *ReadDelegateClientUseCase) Execute(ctx context.Context, req *delegateclientpb.ReadDelegateClientRequest) (*delegateclientpb.ReadDelegateClientResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.id_required", "Delegate-Client relationship ID is required [DEFAULT]"))
	}

	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
		}

		permission := ports.EntityPermission(ports.EntityDelegateClient, ports.ActionRead)
		authorized, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.authorization_check_failed", "Authorization check failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.access_denied", "Access denied [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.ReadDelegateClient(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return empty result for missing entity (following established pattern)
	if len(resp.Data) == 0 {
		return &delegateclientpb.ReadDelegateClientResponse{
			Data:    []*delegateclientpb.DelegateClient{},
			Success: true,
		}, nil
	}

	return resp, nil
}

// Helper functions
