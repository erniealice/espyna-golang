package client_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// UpdateClientAttributeUseCase handles the business logic for updating client attributes
// UpdateClientAttributeRepositories groups all repository dependencies
type UpdateClientAttributeRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
	Client          clientpb.ClientDomainServiceServer                   // Entity reference validation
	Attribute       attributepb.AttributeDomainServiceServer             // Entity reference validation
}

// UpdateClientAttributeServices groups all business service dependencies
type UpdateClientAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateClientAttributeUseCase handles the business logic for updating client attributes
type UpdateClientAttributeUseCase struct {
	repositories UpdateClientAttributeRepositories
	services     UpdateClientAttributeServices
}

// NewUpdateClientAttributeUseCase creates use case with grouped dependencies
func NewUpdateClientAttributeUseCase(
	repositories UpdateClientAttributeRepositories,
	services UpdateClientAttributeServices,
) *UpdateClientAttributeUseCase {
	return &UpdateClientAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateClientAttributeUseCaseUngrouped creates a new UpdateClientAttributeUseCase
// Deprecated: Use NewUpdateClientAttributeUseCase with grouped parameters instead
func NewUpdateClientAttributeUseCaseUngrouped(
	clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateClientAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
		Client:          clientRepo,
		Attribute:       attributeRepo,
	}

	services := UpdateClientAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateClientAttributeUseCase(repositories, services)
}

// Execute performs the update client attribute operation
func (uc *UpdateClientAttributeUseCase) Execute(ctx context.Context, req *clientattributepb.UpdateClientAttributeRequest) (*clientattributepb.UpdateClientAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichClientAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.UpdateClientAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.update_failed", "Client attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateClientAttributeUseCase) validateInput(ctx context.Context, req *clientattributepb.UpdateClientAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "Request is required for client attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.data_required", "Client attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.id_required", "Client attribute ID is required [DEFAULT]"))
	}
	if req.Data.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.client_id_required", "Client ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichClientAttributeData adds updated audit information
func (uc *UpdateClientAttributeUseCase) enrichClientAttributeData(clientAttribute *clientattributepb.ClientAttribute) error {
	now := time.Now()

	// Update modification timestamp
	clientAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	clientAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateClientAttributeUseCase) validateBusinessRules(ctx context.Context, clientAttribute *clientattributepb.ClientAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(clientAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(clientAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Validate client and attribute exist
	// Example: Validate attribute type constraints
	// Example: Check permissions for updating this attribute
	// For now, allow all updates

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateClientAttributeUseCase) validateEntityReferences(ctx context.Context, clientAttribute *clientattributepb.ClientAttribute) error {
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.client_not_found", "Referenced client with ID '{clientId}' does not exist [DEFAULT]")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist [DEFAULT]")
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
