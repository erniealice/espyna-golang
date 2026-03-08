package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// GetAssetItemPageDataRepositories groups all repository dependencies
type GetAssetItemPageDataRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// GetAssetItemPageDataServices groups all business service dependencies
type GetAssetItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetAssetItemPageDataUseCase handles the business logic for getting asset item page data
type GetAssetItemPageDataUseCase struct {
	repositories GetAssetItemPageDataRepositories
	services     GetAssetItemPageDataServices
}

// NewGetAssetItemPageDataUseCase creates use case with grouped dependencies
func NewGetAssetItemPageDataUseCase(
	repositories GetAssetItemPageDataRepositories,
	services GetAssetItemPageDataServices,
) *GetAssetItemPageDataUseCase {
	return &GetAssetItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get asset item page data operation
func (uc *GetAssetItemPageDataUseCase) Execute(ctx context.Context, req *assetpb.GetAssetItemPageDataRequest) (*assetpb.GetAssetItemPageDataResponse, error) {
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
	resp, err := uc.repositories.Asset.GetAssetItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load asset details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetAssetItemPageDataUseCase) validateInput(ctx context.Context, req *assetpb.GetAssetItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate asset ID
	if req.AssetId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.asset_id_required", "[ERR-DEFAULT] Asset ID is required"))
	}

	// Basic ID format validation
	if len(req.AssetId) < 3 || len(req.AssetId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.invalid_asset_id_format", "[ERR-DEFAULT] Invalid asset ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.AssetId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.asset_id_invalid_characters", "[ERR-DEFAULT] Asset ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetAssetItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *assetpb.GetAssetItemPageDataRequest) error {
	// Check authorization for viewing specific asset
	// This would typically involve checking user permissions for the specific asset
	// For now, we'll allow all authenticated users to view asset details

	return nil
}
