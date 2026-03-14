package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// UpdateAssetRepositories groups all repository dependencies
type UpdateAssetRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// UpdateAssetServices groups all business service dependencies
type UpdateAssetServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAssetUseCase handles the business logic for updating assets
type UpdateAssetUseCase struct {
	repositories UpdateAssetRepositories
	services     UpdateAssetServices
}

// NewUpdateAssetUseCase creates use case with grouped dependencies
func NewUpdateAssetUseCase(
	repositories UpdateAssetRepositories,
	services UpdateAssetServices,
) *UpdateAssetUseCase {
	return &UpdateAssetUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *UpdateAssetUseCase) Execute(ctx context.Context, req *assetpb.UpdateAssetRequest) (*assetpb.UpdateAssetResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAsset, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichAssetData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Asset.UpdateAsset(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.update_failed", "[ERR-DEFAULT] Asset update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateAssetUseCase) validateInput(ctx context.Context, req *assetpb.UpdateAssetRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.data_required", "[ERR-DEFAULT] Asset data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.AssetNumber = strings.TrimSpace(req.Data.AssetNumber)
	if req.Data.Description != nil {
		trimmed := strings.TrimSpace(*req.Data.Description)
		req.Data.Description = &trimmed
	}

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.id_required", "[ERR-DEFAULT] Asset ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.AcquisitionCost <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.acquisition_cost_required", "[ERR-DEFAULT] Acquisition cost must be greater than zero"))
	}
	if req.Data.AssetCategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.category_id_required", "[ERR-DEFAULT] Asset category is required"))
	}
	return nil
}

// enrichAssetData adds audit information for updates
func (uc *UpdateAssetUseCase) enrichAssetData(asset *assetpb.Asset) error {
	now := time.Now()

	// Set asset audit fields for modification
	asset.DateModified = &[]int64{now.UnixMilli()}[0]
	asset.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateAssetUseCase) validateBusinessRules(ctx context.Context, asset *assetpb.Asset) error {
	// Validate name length
	if len(asset.Name) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 200 characters"))
	}

	// Validate asset number length
	if len(asset.AssetNumber) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.asset_number_too_long", "[ERR-DEFAULT] Asset number must not exceed 50 characters"))
	}

	// Validate description length if provided
	if asset.Description != nil && len(*asset.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 1000 characters"))
	}

	// Validate salvage value is not negative
	if asset.SalvageValue < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.salvage_value_negative", "[ERR-DEFAULT] Salvage value must not be negative"))
	}

	// Validate salvage value does not exceed acquisition cost
	if asset.SalvageValue > asset.AcquisitionCost {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.salvage_exceeds_cost", "[ERR-DEFAULT] Salvage value must not exceed acquisition cost"))
	}

	// Validate useful life is positive when provided
	if asset.UsefulLifeMonths < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.useful_life_negative", "[ERR-DEFAULT] Useful life must not be negative"))
	}

	return nil
}
