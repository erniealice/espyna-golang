package asset_category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetcategorypb "github.com/erniealice/esqyma/golang/v1/domain/asset/asset_category"
)

// UpdateAssetCategoryRepositories groups all repository dependencies
type UpdateAssetCategoryRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// UpdateAssetCategoryServices groups all business service dependencies
type UpdateAssetCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAssetCategoryUseCase handles the business logic for updating asset categories
type UpdateAssetCategoryUseCase struct {
	repositories UpdateAssetCategoryRepositories
	services     UpdateAssetCategoryServices
}

// NewUpdateAssetCategoryUseCase creates use case with grouped dependencies
func NewUpdateAssetCategoryUseCase(
	repositories UpdateAssetCategoryRepositories,
	services UpdateAssetCategoryServices,
) *UpdateAssetCategoryUseCase {
	return &UpdateAssetCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *UpdateAssetCategoryUseCase) Execute(ctx context.Context, req *assetcategorypb.UpdateAssetCategoryRequest) (*assetcategorypb.UpdateAssetCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichAssetCategoryData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.AssetCategory.UpdateAssetCategory(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.errors.update_failed", "[ERR-DEFAULT] Asset category update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateAssetCategoryUseCase) validateInput(ctx context.Context, req *assetcategorypb.UpdateAssetCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.data_required", "[ERR-DEFAULT] Asset category data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Code = strings.TrimSpace(req.Data.Code)
	if req.Data.Description != nil {
		trimmed := strings.TrimSpace(*req.Data.Description)
		req.Data.Description = &trimmed
	}

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.id_required", "[ERR-DEFAULT] Asset category ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.code_required", "[ERR-DEFAULT] Code is required"))
	}
	return nil
}

// enrichAssetCategoryData adds audit information for updates
func (uc *UpdateAssetCategoryUseCase) enrichAssetCategoryData(category *assetcategorypb.AssetCategory) error {
	now := time.Now()

	// Set audit fields for modification
	category.DateModified = &[]int64{now.UnixMilli()}[0]
	category.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateAssetCategoryUseCase) validateBusinessRules(ctx context.Context, category *assetcategorypb.AssetCategory) error {
	// Validate name length
	if len(category.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	// Validate code length
	if len(category.Code) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.code_too_long", "[ERR-DEFAULT] Code must not exceed 50 characters"))
	}

	// Validate description length if provided
	if category.Description != nil && len(*category.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 1000 characters"))
	}

	// Validate default salvage value percent is between 0 and 100
	if category.DefaultSalvageValuePercent < 0 || category.DefaultSalvageValuePercent > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.salvage_percent_invalid", "[ERR-DEFAULT] Default salvage value percent must be between 0 and 100"))
	}

	// Validate default useful life months is not negative
	if category.DefaultUsefulLifeMonths < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.useful_life_negative", "[ERR-DEFAULT] Default useful life months must not be negative"))
	}

	return nil
}
