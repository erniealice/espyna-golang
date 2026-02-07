package delegate_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// GetDelegateAttributeItemPageDataUseCase handles the business logic for getting delegate attribute item page data
// GetDelegateAttributeItemPageDataRepositories groups all repository dependencies
type GetDelegateAttributeItemPageDataRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
}

// GetDelegateAttributeItemPageDataServices groups all business service dependencies
type GetDelegateAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetDelegateAttributeItemPageDataUseCase handles the business logic for getting delegate attribute item page data
type GetDelegateAttributeItemPageDataUseCase struct {
	repositories GetDelegateAttributeItemPageDataRepositories
	services     GetDelegateAttributeItemPageDataServices
}

// NewGetDelegateAttributeItemPageDataUseCase creates use case with grouped dependencies
func NewGetDelegateAttributeItemPageDataUseCase(
	repositories GetDelegateAttributeItemPageDataRepositories,
	services GetDelegateAttributeItemPageDataServices,
) *GetDelegateAttributeItemPageDataUseCase {
	return &GetDelegateAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetDelegateAttributeItemPageDataUseCaseUngrouped creates a new GetDelegateAttributeItemPageDataUseCase
// Deprecated: Use NewGetDelegateAttributeItemPageDataUseCase with grouped parameters instead
func NewGetDelegateAttributeItemPageDataUseCaseUngrouped(delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer) *GetDelegateAttributeItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetDelegateAttributeItemPageDataRepositories{
		DelegateAttribute: delegateAttributeRepo,
	}

	services := GetDelegateAttributeItemPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetDelegateAttributeItemPageDataUseCase(repositories, services)
}

// Execute performs the get delegate attribute item page data operation
func (uc *GetDelegateAttributeItemPageDataUseCase) Execute(ctx context.Context, req *delegateattributepb.GetDelegateAttributeItemPageDataRequest) (*delegateattributepb.GetDelegateAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.GetDelegateAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.item_page_data_failed", "Failed to retrieve delegate attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetDelegateAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *delegateattributepb.GetDelegateAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", "Request is required for delegate attributes [DEFAULT]"))
	}

	// Validate delegate attribute ID
	if strings.TrimSpace(req.DelegateAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.id_required", "Delegate attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.DelegateAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.id_too_short", "Delegate attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
