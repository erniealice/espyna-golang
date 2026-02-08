package delegate_client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// ListDelegateClientsRepositories groups all repository dependencies
type ListDelegateClientsRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// ListDelegateClientsServices groups all business service dependencies
type ListDelegateClientsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListDelegateClientsUseCase handles the business logic for listing delegate clients
type ListDelegateClientsUseCase struct {
	repositories ListDelegateClientsRepositories
	services     ListDelegateClientsServices
}

// NewListDelegateClientsUseCase creates use case with grouped dependencies
func NewListDelegateClientsUseCase(
	repositories ListDelegateClientsRepositories,
	services ListDelegateClientsServices,
) *ListDelegateClientsUseCase {
	return &ListDelegateClientsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListDelegateClientsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListDelegateClientsUseCase with grouped parameters instead
func NewListDelegateClientsUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *ListDelegateClientsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListDelegateClientsRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       nil, // Not needed for list operations
		Client:         nil, // Not needed for list operations
	}

	services := ListDelegateClientsServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListDelegateClientsUseCase(repositories, services)
}

// Execute performs the list delegate clients operation
func (uc *ListDelegateClientsUseCase) Execute(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) (*delegateclientpb.ListDelegateClientsResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
		}

		// Check permission to list delegate-client relationships
		permission := ports.EntityPermission(ports.EntityDelegateClient, ports.ActionRead)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.authorization_check_failed", "Authorization check failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.access_denied", "Access denied [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}

	// Business logic pre-processing
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.ListDelegateClients(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.list_failed", "Failed to retrieve Delegate-Client relationships [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic post-processing (if needed)
	if err := uc.processListResults(ctx, resp); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.result_processing_failed", "Result processing failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListDelegateClientsUseCase) validateInput(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListDelegateClientsUseCase) validateBusinessRules(ctx context.Context, req *delegateclientpb.ListDelegateClientsRequest) error {
	// Add any business rules for filtering or access control
	// For example, ensure user has permission to view delegate-client relationships

	// Currently no specific business rules for listing
	return nil
}

// processListResults applies any business logic to the list results
func (uc *ListDelegateClientsUseCase) processListResults(ctx context.Context, resp *delegateclientpb.ListDelegateClientsResponse) error {
	// Apply any post-processing business logic
	// For example, filter results based on user permissions
	// or enrich data with additional information

	// Currently no post-processing required
	return nil
}

// Helper functions
