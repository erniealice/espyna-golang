package asset_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
)

// ReadAssetCategoryRepositories groups all repository dependencies
type ReadAssetCategoryRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// ReadAssetCategoryServices groups all business service dependencies
type ReadAssetCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadAssetCategoryUseCase handles the business logic for reading asset categories
type ReadAssetCategoryUseCase struct {
	repositories ReadAssetCategoryRepositories
	services     ReadAssetCategoryServices
}

// NewReadAssetCategoryUseCase creates use case with grouped dependencies
func NewReadAssetCategoryUseCase(
	repositories ReadAssetCategoryRepositories,
	services ReadAssetCategoryServices,
) *ReadAssetCategoryUseCase {
	return &ReadAssetCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ReadAssetCategoryUseCase) Execute(ctx context.Context, req *assetcategorypb.ReadAssetCategoryRequest) (*assetcategorypb.ReadAssetCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.AssetCategory.ReadAssetCategory(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadAssetCategoryUseCase) validateInput(ctx context.Context, req *assetcategorypb.ReadAssetCategoryRequest) error {
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
