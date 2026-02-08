package delegate_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	delegateattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_attribute"
)

// GetDelegateAttributeListPageDataUseCase handles the business logic for getting delegate attribute list page data
// GetDelegateAttributeListPageDataRepositories groups all repository dependencies
type GetDelegateAttributeListPageDataRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
}

// GetDelegateAttributeListPageDataServices groups all business service dependencies
type GetDelegateAttributeListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetDelegateAttributeListPageDataUseCase handles the business logic for getting delegate attribute list page data
type GetDelegateAttributeListPageDataUseCase struct {
	repositories GetDelegateAttributeListPageDataRepositories
	services     GetDelegateAttributeListPageDataServices
}

// NewGetDelegateAttributeListPageDataUseCase creates use case with grouped dependencies
func NewGetDelegateAttributeListPageDataUseCase(
	repositories GetDelegateAttributeListPageDataRepositories,
	services GetDelegateAttributeListPageDataServices,
) *GetDelegateAttributeListPageDataUseCase {
	return &GetDelegateAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetDelegateAttributeListPageDataUseCaseUngrouped creates a new GetDelegateAttributeListPageDataUseCase
// Deprecated: Use NewGetDelegateAttributeListPageDataUseCase with grouped parameters instead
func NewGetDelegateAttributeListPageDataUseCaseUngrouped(delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer) *GetDelegateAttributeListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetDelegateAttributeListPageDataRepositories{
		DelegateAttribute: delegateAttributeRepo,
	}

	services := GetDelegateAttributeListPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetDelegateAttributeListPageDataUseCase(repositories, services)
}

// Execute performs the get delegate attribute list page data operation
func (uc *GetDelegateAttributeListPageDataUseCase) Execute(ctx context.Context, req *delegateattributepb.GetDelegateAttributeListPageDataRequest) (*delegateattributepb.GetDelegateAttributeListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination, filtering, sorting, and search
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.GetDelegateAttributeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.errors.list_page_data_failed", "Failed to retrieve delegate attribute list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetDelegateAttributeListPageDataUseCase) validateInput(ctx context.Context, req *delegateattributepb.GetDelegateAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", "Request is required for delegate attributes [DEFAULT]"))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	// Validate search if provided
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetDelegateAttributeListPageDataUseCase) applyDefaults(req *delegateattributepb.GetDelegateAttributeListPageDataRequest) error {
	// Apply default pagination if not provided
	if req.Pagination == nil {
		req.Pagination = &commonpb.PaginationRequest{
			Limit: 10, // Default page size
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1, // Default to first page
				},
			},
		}
	} else if req.Pagination.Limit == 0 {
		req.Pagination.Limit = 10 // Default page size if not specified
	}

	return nil
}
