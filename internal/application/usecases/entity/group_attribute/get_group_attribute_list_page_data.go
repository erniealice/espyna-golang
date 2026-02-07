package group_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
)

// GetGroupAttributeListPageDataUseCase handles the business logic for getting group attribute list page data
// GetGroupAttributeListPageDataRepositories groups all repository dependencies
type GetGroupAttributeListPageDataRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
}

// GetGroupAttributeListPageDataServices groups all business service dependencies
type GetGroupAttributeListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetGroupAttributeListPageDataUseCase handles the business logic for getting group attribute list page data
type GetGroupAttributeListPageDataUseCase struct {
	repositories GetGroupAttributeListPageDataRepositories
	services     GetGroupAttributeListPageDataServices
}

// NewGetGroupAttributeListPageDataUseCase creates use case with grouped dependencies
func NewGetGroupAttributeListPageDataUseCase(
	repositories GetGroupAttributeListPageDataRepositories,
	services GetGroupAttributeListPageDataServices,
) *GetGroupAttributeListPageDataUseCase {
	return &GetGroupAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetGroupAttributeListPageDataUseCaseUngrouped creates a new GetGroupAttributeListPageDataUseCase
// Deprecated: Use NewGetGroupAttributeListPageDataUseCase with grouped parameters instead
func NewGetGroupAttributeListPageDataUseCaseUngrouped(groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer) *GetGroupAttributeListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetGroupAttributeListPageDataRepositories{
		GroupAttribute: groupAttributeRepo,
	}

	services := GetGroupAttributeListPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetGroupAttributeListPageDataUseCase(repositories, services)
}

// Execute performs the get group attribute list page data operation
func (uc *GetGroupAttributeListPageDataUseCase) Execute(ctx context.Context, req *groupattributepb.GetGroupAttributeListPageDataRequest) (*groupattributepb.GetGroupAttributeListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination, filtering, sorting, and search
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.GetGroupAttributeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.list_page_data_failed", "Failed to retrieve group attribute list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetGroupAttributeListPageDataUseCase) validateInput(ctx context.Context, req *groupattributepb.GetGroupAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "Request is required for group attributes [DEFAULT]"))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	// Validate search if provided
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetGroupAttributeListPageDataUseCase) applyDefaults(req *groupattributepb.GetGroupAttributeListPageDataRequest) error {
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
