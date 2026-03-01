package client_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// CreateClientAttributeRepositories groups all repository dependencies
type CreateClientAttributeRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
	Client          clientpb.ClientDomainServiceServer                   // Entity reference validation
	Attribute       attributepb.AttributeDomainServiceServer             // Entity reference validation
}

// CreateClientAttributeServices groups all business service dependencies
type CreateClientAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateClientAttributeUseCase handles the business logic for creating client attributes
type CreateClientAttributeUseCase struct {
	repositories CreateClientAttributeRepositories
	services     CreateClientAttributeServices
}

// NewCreateClientAttributeUseCase creates use case with grouped dependencies
func NewCreateClientAttributeUseCase(
	repositories CreateClientAttributeRepositories,
	services CreateClientAttributeServices,
) *CreateClientAttributeUseCase {
	return &CreateClientAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateClientAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateClientAttributeUseCase with grouped parameters instead
func NewCreateClientAttributeUseCaseUngrouped(
	clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateClientAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
		Client:          clientRepo,
		Attribute:       attributeRepo,
	}

	services := CreateClientAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateClientAttributeUseCase(repositories, services)
}

// NewCreateClientAttributeUseCaseWithTransaction creates a new CreateClientAttributeUseCase with transaction support
// Deprecated: Use NewCreateClientAttributeUseCase with grouped parameters instead

// Execute performs the create client attribute operation
func (uc *CreateClientAttributeUseCase) Execute(ctx context.Context, req *clientattributepb.CreateClientAttributeRequest) (*clientattributepb.CreateClientAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClientAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichClientAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.CreateClientAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.creation_failed", "Client attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateClientAttributeUseCase) validateInput(ctx context.Context, req *clientattributepb.CreateClientAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.data_required", "[ERR-DEFAULT] Client attribute data is required"))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.client_id_required", "[ERR-DEFAULT] Client ID is required"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.attribute_id_required", "[ERR-DEFAULT] Attribute ID is required"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_required", "[ERR-DEFAULT] Attribute value is required"))
	}
	return nil
}

// enrichClientAttributeData adds generated fields and audit information
func (uc *CreateClientAttributeUseCase) enrichClientAttributeData(clientAttribute *clientattributepb.ClientAttribute) error {
	now := time.Now()

	// Generate ClientAttribute ID
	if clientAttribute.Id == "" {
		clientAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set client attribute audit fields
	clientAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	clientAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	clientAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	clientAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	clientAttribute.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateClientAttributeUseCase) validateBusinessRules(ctx context.Context, clientAttribute *clientattributepb.ClientAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(clientAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(clientAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Check for duplicate client-attribute combinations
	// Example: Validate attribute type constraints
	// For now, allow all combinations

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateClientAttributeUseCase) validateEntityReferences(ctx context.Context, clientAttribute *clientattributepb.ClientAttribute) error {
	// Validate Client entity reference
	if clientAttribute.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: clientAttribute.ClientId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.client_reference_validation_failed", "Failed to validate client entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if client == nil || client.Data == nil || len(client.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.client_not_found", "[ERR-DEFAULT] Client not found")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", clientAttribute.ClientId)
			return errors.New(translatedError)
		}
		if !client.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.client_not_active", "Referenced client with ID '{clientId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{clientId}", clientAttribute.ClientId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if clientAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: clientAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.attribute_not_found", "[ERR-DEFAULT] Attribute not found")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", clientAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", clientAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
