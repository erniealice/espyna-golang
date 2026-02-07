package group_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
)

// UpdateGroupAttributeUseCase handles the business logic for updating group attributes
// UpdateGroupAttributeRepositories groups all repository dependencies
type UpdateGroupAttributeRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
	Group          grouppb.GroupDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// UpdateGroupAttributeServices groups all business service dependencies
type UpdateGroupAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateGroupAttributeUseCase handles the business logic for updating group attributes
type UpdateGroupAttributeUseCase struct {
	repositories UpdateGroupAttributeRepositories
	services     UpdateGroupAttributeServices
}

// NewUpdateGroupAttributeUseCase creates use case with grouped dependencies
func NewUpdateGroupAttributeUseCase(
	repositories UpdateGroupAttributeRepositories,
	services UpdateGroupAttributeServices,
) *UpdateGroupAttributeUseCase {
	return &UpdateGroupAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateGroupAttributeUseCaseUngrouped creates a new UpdateGroupAttributeUseCase
// Deprecated: Use NewUpdateGroupAttributeUseCase with grouped parameters instead
func NewUpdateGroupAttributeUseCaseUngrouped(
	groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer,
	groupRepo grouppb.GroupDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateGroupAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateGroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
		Group:          groupRepo,
		Attribute:      attributeRepo,
	}

	services := UpdateGroupAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateGroupAttributeUseCase(repositories, services)
}

// Execute performs the update group attribute operation
func (uc *UpdateGroupAttributeUseCase) Execute(ctx context.Context, req *groupattributepb.UpdateGroupAttributeRequest) (*groupattributepb.UpdateGroupAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichGroupAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.UpdateGroupAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.update_failed", "Group attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateGroupAttributeUseCase) validateInput(ctx context.Context, req *groupattributepb.UpdateGroupAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "Request is required for group attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.data_required", "Group attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.id_required", "Group attribute ID is required [DEFAULT]"))
	}
	if req.Data.GroupId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.group_id_required", "Group ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichGroupAttributeData adds updated audit information
func (uc *UpdateGroupAttributeUseCase) enrichGroupAttributeData(groupAttribute *groupattributepb.GroupAttribute) error {
	now := time.Now()

	// Update modification timestamp
	groupAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	groupAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateGroupAttributeUseCase) validateBusinessRules(ctx context.Context, groupAttribute *groupattributepb.GroupAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(groupAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(groupAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Validate group and attribute exist
	// Example: Validate attribute type constraints
	// Example: Check permissions for updating this attribute
	// For now, allow all updates

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateGroupAttributeUseCase) validateEntityReferences(ctx context.Context, groupAttribute *groupattributepb.GroupAttribute) error {
	// Validate Group entity reference
	if groupAttribute.GroupId != "" {
		group, err := uc.repositories.Group.ReadGroup(ctx, &grouppb.ReadGroupRequest{
			Data: &grouppb.Group{Id: groupAttribute.GroupId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.group_reference_validation_failed", "Failed to validate group entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if group == nil || group.Data == nil || len(group.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.group_not_found", "Referenced group with ID '{groupId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{groupId}", groupAttribute.GroupId)
			return errors.New(translatedError)
		}
		if !group.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.group_not_active", "Referenced group with ID '{groupId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{groupId}", groupAttribute.GroupId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if groupAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: groupAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", groupAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", groupAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
