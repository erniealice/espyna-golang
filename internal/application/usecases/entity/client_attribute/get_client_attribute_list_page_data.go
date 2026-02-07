package client_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

// GetClientAttributeListPageDataUseCase handles the business logic for getting client attribute list page data
// GetClientAttributeListPageDataRepositories groups all repository dependencies
type GetClientAttributeListPageDataRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
}

// GetClientAttributeListPageDataServices groups all business service dependencies
type GetClientAttributeListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetClientAttributeListPageDataUseCase handles the business logic for getting client attribute list page data
type GetClientAttributeListPageDataUseCase struct {
	repositories GetClientAttributeListPageDataRepositories
	services     GetClientAttributeListPageDataServices
}

// NewGetClientAttributeListPageDataUseCase creates use case with grouped dependencies
func NewGetClientAttributeListPageDataUseCase(
	repositories GetClientAttributeListPageDataRepositories,
	services GetClientAttributeListPageDataServices,
) *GetClientAttributeListPageDataUseCase {
	return &GetClientAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetClientAttributeListPageDataUseCaseUngrouped creates a new GetClientAttributeListPageDataUseCase
// Deprecated: Use NewGetClientAttributeListPageDataUseCase with grouped parameters instead
func NewGetClientAttributeListPageDataUseCaseUngrouped(clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer) *GetClientAttributeListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetClientAttributeListPageDataRepositories{
		ClientAttribute: clientAttributeRepo,
	}

	services := GetClientAttributeListPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetClientAttributeListPageDataUseCase(repositories, services)
}

// Execute performs the get client attribute list page data operation
func (uc *GetClientAttributeListPageDataUseCase) Execute(ctx context.Context, req *clientattributepb.GetClientAttributeListPageDataRequest) (*clientattributepb.GetClientAttributeListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination, filtering, sorting, and search
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.GetClientAttributeListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.list_page_data_failed", "Failed to retrieve client attribute list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetClientAttributeListPageDataUseCase) validateInput(ctx context.Context, req *clientattributepb.GetClientAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "Request is required for client attributes [DEFAULT]"))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	// Validate search if provided
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *GetClientAttributeListPageDataUseCase) applyDefaults(req *clientattributepb.GetClientAttributeListPageDataRequest) error {
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
