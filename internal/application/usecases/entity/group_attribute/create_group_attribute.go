package group_attribute

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
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
	groupattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group_attribute"
)

// CreateGroupAttributeRepositories groups all repository dependencies
type CreateGroupAttributeRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
	Group          grouppb.GroupDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// CreateGroupAttributeServices groups all business service dependencies
type CreateGroupAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateGroupAttributeUseCase handles the business logic for creating group attributes
type CreateGroupAttributeUseCase struct {
	repositories CreateGroupAttributeRepositories
	services     CreateGroupAttributeServices
}

// NewCreateGroupAttributeUseCase creates use case with grouped dependencies
func NewCreateGroupAttributeUseCase(
	repositories CreateGroupAttributeRepositories,
	services CreateGroupAttributeServices,
) *CreateGroupAttributeUseCase {
	return &CreateGroupAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateGroupAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateGroupAttributeUseCase with grouped parameters instead
func NewCreateGroupAttributeUseCaseUngrouped(
	groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer,
	groupRepo grouppb.GroupDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateGroupAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateGroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
		Group:          groupRepo,
		Attribute:      attributeRepo,
	}

	services := CreateGroupAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateGroupAttributeUseCase(repositories, services)
}

// NewCreateGroupAttributeUseCaseWithTransaction creates a new CreateGroupAttributeUseCase with transaction support
// Deprecated: Use NewCreateGroupAttributeUseCase with grouped parameters instead

// Execute performs the create group attribute operation
func (uc *CreateGroupAttributeUseCase) Execute(ctx context.Context, req *groupattributepb.CreateGroupAttributeRequest) (*groupattributepb.CreateGroupAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityGroupAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichGroupAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.CreateGroupAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.creation_failed", "Group attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateGroupAttributeUseCase) validateInput(ctx context.Context, req *groupattributepb.CreateGroupAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.data_required", "[ERR-DEFAULT] Group attribute data is required"))
	}
	if req.Data.GroupId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.group_id_required", "[ERR-DEFAULT] Group ID is required"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.attribute_id_required", "[ERR-DEFAULT] Attribute ID is required"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_required", "[ERR-DEFAULT] Attribute value is required"))
	}
	return nil
}

// enrichGroupAttributeData adds generated fields and audit information
func (uc *CreateGroupAttributeUseCase) enrichGroupAttributeData(groupAttribute *groupattributepb.GroupAttribute) error {
	now := time.Now()

	// Generate GroupAttribute ID
	if groupAttribute.Id == "" {
		groupAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set group attribute audit fields
	groupAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	groupAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	groupAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	groupAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	groupAttribute.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateGroupAttributeUseCase) validateBusinessRules(ctx context.Context, groupAttribute *groupattributepb.GroupAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(groupAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(groupAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Check for duplicate group-attribute combinations
	// Example: Validate attribute type constraints
	// For now, allow all combinations

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateGroupAttributeUseCase) validateEntityReferences(ctx context.Context, groupAttribute *groupattributepb.GroupAttribute) error {
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.group_not_found", "[ERR-DEFAULT] Group not found")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.attribute_not_found", "[ERR-DEFAULT] Attribute not found")
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
