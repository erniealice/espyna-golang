package client_category

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientcategorypb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_category"
)

// GetClientCategoryListPageDataRepositories groups all repository dependencies
type GetClientCategoryListPageDataRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// GetClientCategoryListPageDataServices groups all business service dependencies
type GetClientCategoryListPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetClientCategoryListPageDataUseCase handles the business logic for getting client category list page data
type GetClientCategoryListPageDataUseCase struct {
	repositories GetClientCategoryListPageDataRepositories
	services     GetClientCategoryListPageDataServices
}

// NewGetClientCategoryListPageDataUseCase creates use case with grouped dependencies
func NewGetClientCategoryListPageDataUseCase(
	repositories GetClientCategoryListPageDataRepositories,
	services GetClientCategoryListPageDataServices,
) *GetClientCategoryListPageDataUseCase {
	return &GetClientCategoryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetClientCategoryListPageDataUseCaseUngrouped creates a new GetClientCategoryListPageDataUseCase
// Deprecated: Use NewGetClientCategoryListPageDataUseCase with grouped parameters instead
func NewGetClientCategoryListPageDataUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *GetClientCategoryListPageDataUseCase {
	repositories := GetClientCategoryListPageDataRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := GetClientCategoryListPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetClientCategoryListPageDataUseCase(repositories, services)
}

func (uc *GetClientCategoryListPageDataUseCase) Execute(ctx context.Context, req *clientcategorypb.GetClientCategoryListPageDataRequest) (*clientcategorypb.GetClientCategoryListPageDataResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.ClientCategory.GetClientCategoryListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.errors.list_page_data_failed", "Failed to retrieve client category list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *GetClientCategoryListPageDataUseCase) validateInput(ctx context.Context, req *clientcategorypb.GetClientCategoryListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}

	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.pagination_limit_invalid", "Pagination limit must be non-negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.search_query_too_short", "Search query must be at least 2 characters [DEFAULT]"))
		}
	}

	return nil
}

func (uc *GetClientCategoryListPageDataUseCase) applyDefaults(req *clientcategorypb.GetClientCategoryListPageDataRequest) error {
	if req.Pagination == nil {
		req.Pagination = &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1,
				},
			},
		}
	} else if req.Pagination.Limit == 0 {
		req.Pagination.Limit = 10
	}

	return nil
}
