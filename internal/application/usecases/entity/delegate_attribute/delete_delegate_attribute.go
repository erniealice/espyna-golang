package delegate_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// DeleteDelegateAttributeUseCase handles the business logic for deleting delegate attributes
// DeleteDelegateAttributeRepositories groups all repository dependencies
type DeleteDelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
}

// DeleteDelegateAttributeServices groups all business service dependencies
type DeleteDelegateAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteDelegateAttributeUseCase handles the business logic for deleting delegate attributes
type DeleteDelegateAttributeUseCase struct {
	repositories DeleteDelegateAttributeRepositories
	services     DeleteDelegateAttributeServices
}

// NewDeleteDelegateAttributeUseCase creates use case with grouped dependencies
func NewDeleteDelegateAttributeUseCase(
	repositories DeleteDelegateAttributeRepositories,
	services DeleteDelegateAttributeServices,
) *DeleteDelegateAttributeUseCase {
	return &DeleteDelegateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteDelegateAttributeUseCaseUngrouped creates a new DeleteDelegateAttributeUseCase
// Deprecated: Use NewDeleteDelegateAttributeUseCase with grouped parameters instead
func NewDeleteDelegateAttributeUseCaseUngrouped(delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer) *DeleteDelegateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteDelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
	}

	services := DeleteDelegateAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewDeleteDelegateAttributeUseCase(repositories, services)
}

// Execute performs the delete delegate attribute operation
func (uc *DeleteDelegateAttributeUseCase) Execute(ctx context.Context, req *delegateattributepb.DeleteDelegateAttributeRequest) (*delegateattributepb.DeleteDelegateAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.DeleteDelegateAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.deletion_failed", "Delegate attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteDelegateAttributeUseCase) validateInput(ctx context.Context, req *delegateattributepb.DeleteDelegateAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", "Request is required for delegate attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.data_required", "Delegate attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.id_required", "Delegate attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteDelegateAttributeUseCase) validateBusinessRules(ctx context.Context, req *delegateattributepb.DeleteDelegateAttributeRequest) error {
	// TODO: Additional business rules
	// Example: Check if attribute is required and cannot be deleted
	// Example: Check permissions for deleting this attribute
	// Example: Validate cascading effects
	// For now, allow all deletions

	return nil
}
