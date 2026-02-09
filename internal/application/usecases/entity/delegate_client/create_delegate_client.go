package delegate_client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// CreateDelegateClientRepositories groups all repository dependencies
type CreateDelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// CreateDelegateClientServices groups all business service dependencies
type CreateDelegateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateDelegateClientUseCase handles the business logic for creating delegate clients
type CreateDelegateClientUseCase struct {
	repositories CreateDelegateClientRepositories
	services     CreateDelegateClientServices
}

// NewCreateDelegateClientUseCase creates use case with grouped dependencies
func NewCreateDelegateClientUseCase(
	repositories CreateDelegateClientRepositories,
	services CreateDelegateClientServices,
) *CreateDelegateClientUseCase {
	return &CreateDelegateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateDelegateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateDelegateClientUseCase with grouped parameters instead
func NewCreateDelegateClientUseCaseUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateDelegateClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateDelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       delegateRepo,
		Client:         clientRepo,
	}

	services := CreateDelegateClientServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateDelegateClientUseCase(repositories, services)
}

// NewCreateDelegateClientUseCaseWithTransaction creates a new CreateDelegateClientUseCase with transaction support
// Deprecated: Use NewCreateDelegateClientUseCase with grouped parameters instead

// Execute performs the create delegate client operation
func (uc *CreateDelegateClientUseCase) Execute(ctx context.Context, req *delegateclientpb.CreateDelegateClientRequest) (*delegateclientpb.CreateDelegateClientResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegateClient, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichDelegateClientData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.DelegateClient.CreateDelegateClient(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.errors.creation_failed", "Delegate-Client relationship creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateDelegateClientUseCase) validateInput(ctx context.Context, req *delegateclientpb.CreateDelegateClientRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.request_required", "Request is required for Delegate-Client relationships [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.data_required", "Delegate-Client relationship data is required [DEFAULT]"))
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
func (uc *CreateDelegateClientUseCase) enrichDelegateClientData(delegateClient *delegateclientpb.DelegateClient) error {
	now := time.Now()

	// Generate DelegateClient ID if not provided
	if delegateClient.Id == "" {
		delegateClient.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	delegateClient.DateCreated = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	delegateClient.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	delegateClient.DateModified = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	delegateClient.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	delegateClient.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateDelegateClientUseCase) validateBusinessRules(ctx context.Context, delegateClient *delegateclientpb.DelegateClient) error {
	// Validate delegate and client relationship
	if delegateClient.DelegateId == delegateClient.ClientId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_client.validation.same_ids_not_allowed", "Delegate ID and client ID cannot be the same [DEFAULT]"))
	}

	// Business rule: Prevent duplicate delegate-client relationships
	// This validation should be checked at the repository level to ensure uniqueness
	// The repository implementation should check if a relationship already exists
	// between the delegate and client before creating a new one

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateDelegateClientUseCase) validateEntityReferences(ctx context.Context, delegateClient *delegateclientpb.DelegateClient) error {
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
