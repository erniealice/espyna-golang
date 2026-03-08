package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// DeleteAssetRepositories groups all repository dependencies
type DeleteAssetRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// DeleteAssetServices groups all business service dependencies
type DeleteAssetServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteAssetUseCase handles the business logic for deleting assets
type DeleteAssetUseCase struct {
	repositories DeleteAssetRepositories
	services     DeleteAssetServices
}

// NewDeleteAssetUseCase creates use case with grouped dependencies
func NewDeleteAssetUseCase(
	repositories DeleteAssetRepositories,
	services DeleteAssetServices,
) *DeleteAssetUseCase {
	return &DeleteAssetUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete asset operation
func (uc *DeleteAssetUseCase) Execute(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAsset, ports.ActionDelete); err != nil {
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
	resp, err := uc.repositories.Asset.DeleteAsset(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.deletion_failed", "[ERR-DEFAULT] Asset deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteAssetUseCase) validateInput(ctx context.Context, req *assetpb.DeleteAssetRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteAssetUseCase) validateBusinessRules(ctx context.Context, req *assetpb.DeleteAssetRequest) error {
	// TODO: Add business rules for asset deletion
	// Example: Check if asset has active depreciation schedules, maintenance records, etc.
	// For now, allow all deletions

	return nil
}
