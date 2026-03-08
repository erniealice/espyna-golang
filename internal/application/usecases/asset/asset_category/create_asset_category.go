package asset_category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
)

const entityAssetCategory = "asset_category"

// CreateAssetCategoryRepositories groups all repository dependencies
type CreateAssetCategoryRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// CreateAssetCategoryServices groups all business service dependencies
type CreateAssetCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAssetCategoryUseCase handles the business logic for creating asset categories
type CreateAssetCategoryUseCase struct {
	repositories CreateAssetCategoryRepositories
	services     CreateAssetCategoryServices
}

// NewCreateAssetCategoryUseCase creates use case with grouped dependencies
func NewCreateAssetCategoryUseCase(
	repositories CreateAssetCategoryRepositories,
	services CreateAssetCategoryServices,
) *CreateAssetCategoryUseCase {
	return &CreateAssetCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateAssetCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateAssetCategoryUseCase with grouped parameters instead
func NewCreateAssetCategoryUseCaseUngrouped(assetCategoryRepo assetcategorypb.AssetCategoryDomainServiceServer) *CreateAssetCategoryUseCase {
	repositories := CreateAssetCategoryRepositories{
		AssetCategory: assetCategoryRepo,
	}

	services := CreateAssetCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateAssetCategoryUseCase(repositories, services)
}

// Execute performs the create asset category operation
func (uc *CreateAssetCategoryUseCase) Execute(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes asset category creation within a transaction
func (uc *CreateAssetCategoryUseCase) executeWithTransaction(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
	var result *assetcategorypb.CreateAssetCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "asset_category.errors.creation_failed", "Asset category creation failed [DEFAULT]")
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
func (uc *CreateAssetCategoryUseCase) executeCore(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichAssetCategoryData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.AssetCategory.CreateAssetCategory(ctx, req)
}

// validateInput validates the input request
func (uc *CreateAssetCategoryUseCase) validateInput(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) error {
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

	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset_category.validation.code_required", "[ERR-DEFAULT] Code is required"))
	}
	return nil
}

// enrichAssetCategoryData adds generated fields and audit information
func (uc *CreateAssetCategoryUseCase) enrichAssetCategoryData(category *assetcategorypb.AssetCategory) error {
	now := time.Now()

	// Generate AssetCategory ID if not provided
	if category.Id == "" {
		category.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	category.DateCreated = &[]int64{now.UnixMilli()}[0]
	category.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	category.DateModified = &[]int64{now.UnixMilli()}[0]
	category.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	category.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateAssetCategoryUseCase) validateBusinessRules(ctx context.Context, category *assetcategorypb.AssetCategory) error {
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
