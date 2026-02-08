package delegate_client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// UpdateDelegateClientRepositories groups all repository dependencies
type UpdateDelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// UpdateDelegateClientServices groups all business service dependencies
type UpdateDelegateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateDelegateClientUseCase handles the business logic for updating delegate clients
type UpdateDelegateClientUseCase struct {
	repositories UpdateDelegateClientRepositories
	services     UpdateDelegateClientServices
}

// NewUpdateDelegateClientUseCase creates use case with grouped dependencies
func NewUpdateDelegateClientUseCase(
	repositories UpdateDelegateClientRepositories,
	services UpdateDelegateClientServices,
) *UpdateDelegateClientUseCase {
	return &UpdateDelegateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateDelegateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateDelegateClientUseCase with grouped parameters instead
func NewUpdateDelegateClientUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UpdateDelegateClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateDelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       delegateRepo,
		Client:         clientRepo,
	}

	services := UpdateDelegateClientServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateDelegateClientUseCase(repositories, services)
}

// Execute performs the update delegate client operation
func (uc *UpdateDelegateClientUseCase) Execute(ctx context.Context, req *delegateclientpb.UpdateDelegateClientRequest) (*delegateclientpb.UpdateDelegateClientResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
		}

		// Check permission to update delegate-client relationships
		permission := ports.EntityPermission(ports.EntityDelegateClient, ports.ActionUpdate)
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
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichDelegateClientData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.UpdateDelegateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.update_failed", "Delegate-Client relationship update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateDelegateClientUseCase) validateInput(ctx context.Context, req *delegateclientpb.UpdateDelegateClientRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.data_required", "Delegate-Client relationship data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.id_required", "Delegate-Client relationship ID is required [DEFAULT]"))
	}
	if req.Data.DelegateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.delegate_id_required", "Delegate ID is required [DEFAULT]"))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.client_id_required", "Client ID is required [DEFAULT]"))
	}
	return nil
}

// enrichDelegateClientData adds generated fields and audit information
func (uc *UpdateDelegateClientUseCase) enrichDelegateClientData(delegateClient *delegateclientpb.DelegateClient) error {
	now := time.Now()

	// Update audit fields (preserve original creation date)
	delegateClient.DateModified = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	delegateClient.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateDelegateClientUseCase) validateBusinessRules(ctx context.Context, delegateClient *delegateclientpb.DelegateClient) error {
	// Validate delegate and client relationship
	if delegateClient.DelegateId == delegateClient.ClientId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.same_id", "Delegate ID and client ID cannot be the same [DEFAULT]"))
	}

	// Business rule: Prevent duplicate delegate-client relationships
	// This validation should be checked at the repository level to ensure uniqueness
	// The repository implementation should check if a relationship already exists
	// between the delegate and client (excluding the current record being updated)

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateDelegateClientUseCase) validateEntityReferences(ctx context.Context, delegateClient *delegateclientpb.DelegateClient) error {
	// Validate Delegate entity reference
	if delegateClient.DelegateId != "" {
		delegate, err := uc.repositories.Delegate.ReadDelegate(ctx, &delegatepb.ReadDelegateRequest{
			Data: &delegatepb.Delegate{Id: delegateClient.DelegateId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.delegate_reference_validation_failed", "Failed to validate delegate entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if delegate == nil || delegate.Data == nil || len(delegate.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.delegate_not_found", "Referenced delegate with ID '{delegateId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{delegateId}", delegateClient.DelegateId)
			return errors.New(translatedError)
		}
		if !delegate.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.delegate_not_active", "Referenced delegate with ID '{delegateId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{delegateId}", delegateClient.DelegateId)
			return errors.New(translatedError)
		}
	}

	// Validate Client entity reference
	if delegateClient.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: delegateClient.ClientId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.client_reference_validation_failed", "Failed to validate client entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if client == nil || client.Data == nil || len(client.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.client_not_found", "Referenced client with ID '{clientId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", delegateClient.ClientId)
			return errors.New(translatedError)
		}
		if !client.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.client_not_active", "Referenced client with ID '{clientId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", delegateClient.ClientId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions

// Additional validation methods can be added here as needed
