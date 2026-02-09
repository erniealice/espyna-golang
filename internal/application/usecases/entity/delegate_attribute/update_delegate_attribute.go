package delegate_attribute

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
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_attribute"
)

// UpdateDelegateAttributeUseCase handles the business logic for updating delegate attributes
// UpdateDelegateAttributeRepositories groups all repository dependencies
type UpdateDelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
	Delegate          delegatepb.DelegateDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// UpdateDelegateAttributeServices groups all business service dependencies
type UpdateDelegateAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateDelegateAttributeUseCase handles the business logic for updating delegate attributes
type UpdateDelegateAttributeUseCase struct {
	repositories UpdateDelegateAttributeRepositories
	services     UpdateDelegateAttributeServices
}

// NewUpdateDelegateAttributeUseCase creates use case with grouped dependencies
func NewUpdateDelegateAttributeUseCase(
	repositories UpdateDelegateAttributeRepositories,
	services UpdateDelegateAttributeServices,
) *UpdateDelegateAttributeUseCase {
	return &UpdateDelegateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateDelegateAttributeUseCaseUngrouped creates a new UpdateDelegateAttributeUseCase
// Deprecated: Use NewUpdateDelegateAttributeUseCase with grouped parameters instead
func NewUpdateDelegateAttributeUseCaseUngrouped(
	delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateDelegateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateDelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
		Delegate:          delegateRepo,
		Attribute:         attributeRepo,
	}

	services := UpdateDelegateAttributeServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateDelegateAttributeUseCase(repositories, services)
}

// Execute performs the update delegate attribute operation
func (uc *UpdateDelegateAttributeUseCase) Execute(ctx context.Context, req *delegateattributepb.UpdateDelegateAttributeRequest) (*delegateattributepb.UpdateDelegateAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegateAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichDelegateAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.UpdateDelegateAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.update_failed", "Delegate attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateDelegateAttributeUseCase) validateInput(ctx context.Context, req *delegateattributepb.UpdateDelegateAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", "Request is required for delegate attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.data_required", "Delegate attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.id_required", "Delegate attribute ID is required [DEFAULT]"))
	}
	if req.Data.DelegateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.delegate_id_required", "Delegate ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichDelegateAttributeData adds updated audit information
func (uc *UpdateDelegateAttributeUseCase) enrichDelegateAttributeData(delegateAttribute *delegateattributepb.DelegateAttribute) error {
	now := time.Now()

	// Update modification timestamp
	delegateAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	delegateAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateDelegateAttributeUseCase) validateBusinessRules(ctx context.Context, delegateAttribute *delegateattributepb.DelegateAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(delegateAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(delegateAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Validate delegate and attribute exist
	// Example: Validate attribute type constraints
	// Example: Check permissions for updating this attribute
	// For now, allow all updates

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateDelegateAttributeUseCase) validateEntityReferences(ctx context.Context, delegateAttribute *delegateattributepb.DelegateAttribute) error {
	// Validate Delegate entity reference
	if delegateAttribute.DelegateId != "" {
		delegate, err := uc.repositories.Delegate.ReadDelegate(ctx, &delegatepb.ReadDelegateRequest{
			Data: &delegatepb.Delegate{Id: delegateAttribute.DelegateId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_reference_validation_failed", "Failed to validate delegate entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if delegate == nil || delegate.Data == nil || len(delegate.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_not_found", "Referenced delegate with ID '{delegateId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{delegateId}", delegateAttribute.DelegateId)
			return errors.New(translatedError)
		}
		if !delegate.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_not_active", "Referenced delegate with ID '{delegateId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{delegateId}", delegateAttribute.DelegateId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if delegateAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: delegateAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", delegateAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", delegateAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
