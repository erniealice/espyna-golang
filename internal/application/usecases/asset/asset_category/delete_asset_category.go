package asset_category

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetcategorypb "github.com/erniealice/esqyma/golang/v1/domain/asset/asset_category"
)

// DeleteAssetCategoryRepositories groups all repository dependencies
type DeleteAssetCategoryRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// DeleteAssetCategoryServices groups all business service dependencies
type DeleteAssetCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteAssetCategoryUseCase handles the business logic for deleting asset categories
type DeleteAssetCategoryUseCase struct {
	repositories DeleteAssetCategoryRepositories
	services     DeleteAssetCategoryServices
}

// NewDeleteAssetCategoryUseCase creates use case with grouped dependencies
func NewDeleteAssetCategoryUseCase(
	repositories DeleteAssetCategoryRepositories,
	services DeleteAssetCategoryServices,
) *DeleteAssetCategoryUseCase {
	return &DeleteAssetCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete asset category operation
func (uc *DeleteAssetCategoryUseCase) Execute(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) (*assetcategorypb.DeleteAssetCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionDelete); err != nil {
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
	resp, err := uc.repositories.AssetCategory.DeleteAssetCategory(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.deletion_failed", "[ERR-DEFAULT] Asset category deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteAssetCategoryUseCase) validateInput(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteAssetCategoryUseCase) validateBusinessRules(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) error {
	// TODO: Add business rules for asset category deletion
	// Example: Check if category has associated assets
	// For now, allow all deletions

	return nil
}
