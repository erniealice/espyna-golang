package asset_category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetcategorypb "github.com/erniealice/esqyma/golang/v1/domain/asset/asset_category"
)

// GetAssetCategoryItemPageDataRepositories groups all repository dependencies
type GetAssetCategoryItemPageDataRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// GetAssetCategoryItemPageDataServices groups all business service dependencies
type GetAssetCategoryItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAssetCategoryItemPageDataUseCase handles the business logic for getting asset category item page data
type GetAssetCategoryItemPageDataUseCase struct {
	repositories GetAssetCategoryItemPageDataRepositories
	services     GetAssetCategoryItemPageDataServices
}

// NewGetAssetCategoryItemPageDataUseCase creates use case with grouped dependencies
func NewGetAssetCategoryItemPageDataUseCase(
	repositories GetAssetCategoryItemPageDataRepositories,
	services GetAssetCategoryItemPageDataServices,
) *GetAssetCategoryItemPageDataUseCase {
	return &GetAssetCategoryItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get asset category item page data operation
func (uc *GetAssetCategoryItemPageDataUseCase) Execute(ctx context.Context, req *assetcategorypb.GetAssetCategoryItemPageDataRequest) (*assetcategorypb.GetAssetCategoryItemPageDataResponse, error) {
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
	resp, err := uc.repositories.AssetCategory.GetAssetCategoryItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load asset category details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetAssetCategoryItemPageDataUseCase) validateInput(ctx context.Context, req *assetcategorypb.GetAssetCategoryItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate asset category ID
	if req.AssetCategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.asset_category_id_required", "[ERR-DEFAULT] Asset category ID is required"))
	}

	// Basic ID format validation
	if len(req.AssetCategoryId) < 3 || len(req.AssetCategoryId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.invalid_asset_category_id_format", "[ERR-DEFAULT] Invalid asset category ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.AssetCategoryId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.asset_category_id_invalid_characters", "[ERR-DEFAULT] Asset category ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetAssetCategoryItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *assetcategorypb.GetAssetCategoryItemPageDataRequest) error {
	// Check authorization for viewing specific asset category
	// This would typically involve checking user permissions for the specific asset category
	// For now, we'll allow all authenticated users to view asset category details

	return nil
}
