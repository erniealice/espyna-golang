package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// GetAssetListPageDataRepositories groups all repository dependencies
type GetAssetListPageDataRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// GetAssetListPageDataServices groups all business service dependencies
type GetAssetListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAssetListPageDataUseCase handles the business logic for getting asset list page data with pagination, filtering, sorting, and search
type GetAssetListPageDataUseCase struct {
	repositories GetAssetListPageDataRepositories
	services     GetAssetListPageDataServices
}

// NewGetAssetListPageDataUseCase creates use case with grouped dependencies
func NewGetAssetListPageDataUseCase(
	repositories GetAssetListPageDataRepositories,
	services GetAssetListPageDataServices,
) *GetAssetListPageDataUseCase {
	return &GetAssetListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get asset list page data operation
func (uc *GetAssetListPageDataUseCase) Execute(ctx context.Context, req *assetpb.GetAssetListPageDataRequest) (*assetpb.GetAssetListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAsset, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Asset.GetAssetListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load asset list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetAssetListPageDataUseCase) validateInput(ctx context.Context, req *assetpb.GetAssetListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting list page data
func (uc *GetAssetListPageDataUseCase) validateBusinessRules(ctx context.Context, req *assetpb.GetAssetListPageDataRequest) error {
	// Check authorization for viewing assets
	// This would typically involve checking user permissions
	// For now, we'll allow all authenticated users to view asset lists

	return nil
}
