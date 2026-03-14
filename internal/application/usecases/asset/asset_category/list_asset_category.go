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

// ListAssetCategoriesRepositories groups all repository dependencies
type ListAssetCategoriesRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// ListAssetCategoriesServices groups all business service dependencies
type ListAssetCategoriesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAssetCategoriesUseCase handles the business logic for listing asset categories
type ListAssetCategoriesUseCase struct {
	repositories ListAssetCategoriesRepositories
	services     ListAssetCategoriesServices
}

// NewListAssetCategoriesUseCase creates use case with grouped dependencies
func NewListAssetCategoriesUseCase(
	repositories ListAssetCategoriesRepositories,
	services ListAssetCategoriesServices,
) *ListAssetCategoriesUseCase {
	return &ListAssetCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list asset categories operation
func (uc *ListAssetCategoriesUseCase) Execute(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) (*assetcategorypb.ListAssetCategoriesResponse, error) {
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
	resp, err := uc.repositories.AssetCategory.ListAssetCategories(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.list_failed", "[ERR-DEFAULT] Failed to list asset categories")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListAssetCategoriesUseCase) validateInput(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListAssetCategoriesUseCase) validateBusinessRules(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) error {
	// No additional business rules for listing asset categories
	return nil
}
