package delegate_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// CreateDelegateAttributeRepositories groups all repository dependencies
type CreateDelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
	Delegate          delegatepb.DelegateDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// CreateDelegateAttributeServices groups all business service dependencies
type CreateDelegateAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateDelegateAttributeUseCase handles the business logic for creating delegate attributes
type CreateDelegateAttributeUseCase struct {
	repositories CreateDelegateAttributeRepositories
	services     CreateDelegateAttributeServices
}

// NewCreateDelegateAttributeUseCase creates use case with grouped dependencies
func NewCreateDelegateAttributeUseCase(
	repositories CreateDelegateAttributeRepositories,
	services CreateDelegateAttributeServices,
) *CreateDelegateAttributeUseCase {
	return &CreateDelegateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateDelegateAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateDelegateAttributeUseCase with grouped parameters instead
func NewCreateDelegateAttributeUseCaseUngrouped(
	delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateDelegateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateDelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
		Delegate:          delegateRepo,
		Attribute:         attributeRepo,
	}

	services := CreateDelegateAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateDelegateAttributeUseCase(repositories, services)
}

// NewCreateDelegateAttributeUseCaseWithTransaction creates a new CreateDelegateAttributeUseCase with transaction support
// Deprecated: Use NewCreateDelegateAttributeUseCase with grouped parameters instead

// Execute performs the create delegate attribute operation
func (uc *CreateDelegateAttributeUseCase) Execute(ctx context.Context, req *delegateattributepb.CreateDelegateAttributeRequest) (*delegateattributepb.CreateDelegateAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check (after input validation)
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
		}

		// Note: For delegate attributes, we need to get workspace from delegate
		delegate, err := uc.repositories.Delegate.ReadDelegate(ctx, &delegatepb.ReadDelegateRequest{
			Data: &delegatepb.Delegate{Id: req.Data.DelegateId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_fetch_failed_auth", "Failed to get delegate for authorization [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
		if delegate == nil || delegate.Data == nil || len(delegate.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_not_found", ""))
		}

		// TODO: Re-enable workspace-scoped authorization check once Delegate.WorkspaceId is available
		// workspaceID := delegate.Data[0].WorkspaceId
		// if workspaceID == "" {
		// 	return nil, fmt.Errorf("workspace ID is required for authorization check")
		// }

		// permission := ports.EntityPermission(ports.EntityDelegateAttribute, ports.ActionCreate)
		// authorized, err := uc.services.AuthorizationService.HasPermissionInWorkspace(ctx, userID, workspaceID, permission)
		// if err != nil {
		// 	return nil, fmt.Errorf("authorization check failed: %w", err)
		// }

		// if !authorized {
		// 	return nil, ports.ErrWorkspaceAccessDenied(userID, workspaceID).WithDetails("permission", permission)
		// }
	}

	// Business logic and enrichment
	if err := uc.enrichDelegateAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.CreateDelegateAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.creation_failed", "Delegate attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateDelegateAttributeUseCase) validateInput(ctx context.Context, req *delegateattributepb.CreateDelegateAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.data_required", ""))
	}
	if req.Data.DelegateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.delegate_id_required", ""))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.attribute_id_required", ""))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_required", ""))
	}
	return nil
}

// enrichDelegateAttributeData adds generated fields and audit information
func (uc *CreateDelegateAttributeUseCase) enrichDelegateAttributeData(delegateAttribute *delegateattributepb.DelegateAttribute) error {
	now := time.Now()

	// Generate DelegateAttribute ID
	if delegateAttribute.Id == "" {
		delegateAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set delegate attribute audit fields
	delegateAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	delegateAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	delegateAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	delegateAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	delegateAttribute.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateDelegateAttributeUseCase) validateBusinessRules(ctx context.Context, delegateAttribute *delegateattributepb.DelegateAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(delegateAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(delegateAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Check for duplicate delegate-attribute combinations
	// Example: Validate attribute type constraints
	// For now, allow all combinations

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateDelegateAttributeUseCase) validateEntityReferences(ctx context.Context, delegateAttribute *delegateattributepb.DelegateAttribute) error {
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.delegate_not_found", "")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.attribute_not_found", "")
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
