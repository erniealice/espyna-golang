package delegate_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// ListDelegateAttributesUseCase handles the business logic for listing delegate attributes
// ListDelegateAttributesRepositories groups all repository dependencies
type ListDelegateAttributesRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
}

// ListDelegateAttributesServices groups all business service dependencies
type ListDelegateAttributesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListDelegateAttributesUseCase handles the business logic for listing delegate attributes
type ListDelegateAttributesUseCase struct {
	repositories ListDelegateAttributesRepositories
	services     ListDelegateAttributesServices
}

// NewListDelegateAttributesUseCase creates use case with grouped dependencies
func NewListDelegateAttributesUseCase(
	repositories ListDelegateAttributesRepositories,
	services ListDelegateAttributesServices,
) *ListDelegateAttributesUseCase {
	return &ListDelegateAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListDelegateAttributesUseCaseUngrouped creates a new ListDelegateAttributesUseCase
// Deprecated: Use NewListDelegateAttributesUseCase with grouped parameters instead
func NewListDelegateAttributesUseCaseUngrouped(delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer) *ListDelegateAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListDelegateAttributesRepositories{
		DelegateAttribute: delegateAttributeRepo,
	}

	services := ListDelegateAttributesServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListDelegateAttributesUseCase(repositories, services)
}

// Execute performs the list delegate attributes operation
func (uc *ListDelegateAttributesUseCase) Execute(ctx context.Context, req *delegateattributepb.ListDelegateAttributesRequest) (*delegateattributepb.ListDelegateAttributesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.ListDelegateAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.list_failed", "Failed to retrieve delegate attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListDelegateAttributesUseCase) validateInput(ctx context.Context, req *delegateattributepb.ListDelegateAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", "Request is required for delegate attributes [DEFAULT]"))
	}

	// No additional business rules for listing delegate attributes
	// Pagination is not supported in current protobuf definition

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *ListDelegateAttributesUseCase) applyDefaults(req *delegateattributepb.ListDelegateAttributesRequest) error {
	// No defaults to apply
	// Pagination is not supported in current protobuf definition
	return nil
}
