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

const entityAsset = "asset"

// CreateAssetRepositories groups all repository dependencies
type CreateAssetRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// CreateAssetServices groups all business service dependencies
type CreateAssetServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAssetUseCase handles the business logic for creating assets
type CreateAssetUseCase struct {
	repositories CreateAssetRepositories
	services     CreateAssetServices
}

// NewCreateAssetUseCase creates use case with grouped dependencies
func NewCreateAssetUseCase(
	repositories CreateAssetRepositories,
	services CreateAssetServices,
) *CreateAssetUseCase {
	return &CreateAssetUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateAssetUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateAssetUseCase with grouped parameters instead
func NewCreateAssetUseCaseUngrouped(assetRepo assetpb.AssetDomainServiceServer) *CreateAssetUseCase {
	repositories := CreateAssetRepositories{
		Asset: assetRepo,
	}

	services := CreateAssetServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateAssetUseCase(repositories, services)
}

// Execute performs the create asset operation
func (uc *CreateAssetUseCase) Execute(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAsset, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes asset creation within a transaction
func (uc *CreateAssetUseCase) executeWithTransaction(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
	var result *assetpb.CreateAssetResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "asset.errors.creation_failed", "Asset creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *CreateAssetUseCase) executeCore(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichAssetData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Asset.CreateAsset(ctx, req)
}

// validateInput validates the input request
func (uc *CreateAssetUseCase) validateInput(ctx context.Context, req *assetpb.CreateAssetRequest) error {
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

// enrichAssetData adds generated fields and audit information
func (uc *CreateAssetUseCase) enrichAssetData(asset *assetpb.Asset) error {
	now := time.Now()

	// Generate Asset ID if not provided
	if asset.Id == "" {
		asset.Id = uc.services.IDService.GenerateID()
	}

	// Set asset audit fields
	asset.DateCreated = &[]int64{now.UnixMilli()}[0]
	asset.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	asset.DateModified = &[]int64{now.UnixMilli()}[0]
	asset.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	asset.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateAssetUseCase) validateBusinessRules(ctx context.Context, asset *assetpb.Asset) error {
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
