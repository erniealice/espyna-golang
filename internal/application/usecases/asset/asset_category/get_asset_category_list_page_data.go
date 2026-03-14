package asset_category

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
)

// GetAssetCategoryListPageDataRepositories groups all repository dependencies
type GetAssetCategoryListPageDataRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// GetAssetCategoryListPageDataServices groups all business service dependencies
type GetAssetCategoryListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAssetCategoryListPageDataUseCase handles the business logic for getting asset category list page data with pagination, filtering, sorting, and search
type GetAssetCategoryListPageDataUseCase struct {
	repositories GetAssetCategoryListPageDataRepositories
	services     GetAssetCategoryListPageDataServices
}

// NewGetAssetCategoryListPageDataUseCase creates use case with grouped dependencies
func NewGetAssetCategoryListPageDataUseCase(
	repositories GetAssetCategoryListPageDataRepositories,
	services GetAssetCategoryListPageDataServices,
) *GetAssetCategoryListPageDataUseCase {
	return &GetAssetCategoryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get asset category list page data operation
func (uc *GetAssetCategoryListPageDataUseCase) Execute(ctx context.Context, req *assetcategorypb.GetAssetCategoryListPageDataRequest) (*assetcategorypb.GetAssetCategoryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.AssetCategory.GetAssetCategoryListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load asset category list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetAssetCategoryListPageDataUseCase) validateInput(ctx context.Context, req *assetcategorypb.GetAssetCategoryListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetAssetCategoryListPageDataUseCase) validateBusinessRules(ctx context.Context, req *assetcategorypb.GetAssetCategoryListPageDataRequest) error {
	// Check authorization for viewing asset categories
	// This would typically involve checking user permissions
	// For now, we'll allow all authenticated users to view asset category lists

	return nil
}
