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

// AssetCategoryWithRollup enriches an AssetCategory with per-policy aggregate counts.
type AssetCategoryWithRollup struct {
	Category        *assetcategorypb.AssetCategory
	AssetsInPolicy  int // assets with status IN_SERVICE in this category
	AssetsDeviating int // in-service assets whose depreciation config deviates from category defaults
}

// PolicyRollupRepository extends AssetCategoryDomainServiceServer with the
// rollup aggregation method. Postgres adapters implement this interface by
// running a single JOIN-aggregate query.
type PolicyRollupRepository interface {
	assetcategorypb.AssetCategoryDomainServiceServer

	// ListAssetCategoriesWithPolicyRollup returns all active asset categories
	// with per-category IN_SERVICE asset counts and deviating-asset counts.
	// workspace_id is derived from ctx by the workspace-aware adapter layer.
	ListAssetCategoriesWithPolicyRollup(ctx context.Context) ([]AssetCategoryWithRollup, error)
}

// ListAssetCategoriesWithPolicyRollupRepositories groups all repository dependencies.
type ListAssetCategoriesWithPolicyRollupRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer
}

// ListAssetCategoriesWithPolicyRollupServices groups all service dependencies.
type ListAssetCategoriesWithPolicyRollupServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAssetCategoriesWithPolicyRollupUseCase handles listing categories with rollup counts.
type ListAssetCategoriesWithPolicyRollupUseCase struct {
	repositories ListAssetCategoriesWithPolicyRollupRepositories
	services     ListAssetCategoriesWithPolicyRollupServices
}

// NewListAssetCategoriesWithPolicyRollupUseCase creates the use case with grouped dependencies.
func NewListAssetCategoriesWithPolicyRollupUseCase(
	repositories ListAssetCategoriesWithPolicyRollupRepositories,
	services ListAssetCategoriesWithPolicyRollupServices,
) *ListAssetCategoriesWithPolicyRollupUseCase {
	return &ListAssetCategoriesWithPolicyRollupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list-with-rollup operation.
//
// If the underlying repository implements PolicyRollupRepository, a single
// bulk query is executed (preferred). Otherwise falls back to listing categories
// only (counts = 0), so callers never receive an error simply because the
// rollup SQL extension isn't available.
func (uc *ListAssetCategoriesWithPolicyRollupUseCase) Execute(
	ctx context.Context,
	_ *assetcategorypb.ListAssetCategoriesWithPolicyRollupRequest,
) (*assetcategorypb.ListAssetCategoriesWithPolicyRollupResponse, error) {
	// Authorization check — same permission as list.
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetCategory, ports.ActionList); err != nil {
		return nil, err
	}

	if uc.repositories.AssetCategory == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"asset_category.errors.repository_unavailable", "[ERR-DEFAULT] Repository unavailable"))
	}

	// Preferred path: adapter implements the rollup extension.
	if rollupRepo, ok := uc.repositories.AssetCategory.(PolicyRollupRepository); ok {
		rows, err := rollupRepo.ListAssetCategoriesWithPolicyRollup(ctx)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
				"asset_category.errors.rollup_query_failed", "[ERR-DEFAULT] Failed to load depreciation policy rollup")
			return nil, fmt.Errorf("%s: %w", msg, err)
		}
		return goRollupToProto(rows), nil
	}

	// Fallback path: adapter doesn't support rollup — list categories with zero counts.
	// This gracefully degrades instead of failing; counts will show 0 until the
	// postgresql build tag is active and the adapter is upgraded.
	listResp, err := uc.repositories.AssetCategory.ListAssetCategories(ctx, &assetcategorypb.ListAssetCategoriesRequest{})
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"asset_category.errors.list_failed", "[ERR-DEFAULT] Failed to list asset categories")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	rows := make([]AssetCategoryWithRollup, 0, len(listResp.GetData()))
	for _, cat := range listResp.GetData() {
		rows = append(rows, AssetCategoryWithRollup{Category: cat})
	}
	return goRollupToProto(rows), nil
}

// goRollupToProto converts the internal Go-struct slice to the proto response.
func goRollupToProto(rows []AssetCategoryWithRollup) *assetcategorypb.ListAssetCategoriesWithPolicyRollupResponse {
	items := make([]*assetcategorypb.AssetCategoryWithPolicyRollup, 0, len(rows))
	for _, r := range rows {
		items = append(items, &assetcategorypb.AssetCategoryWithPolicyRollup{
			Category:        r.Category,
			AssetsInPolicy:  int32(r.AssetsInPolicy),
			AssetsDeviating: int32(r.AssetsDeviating),
		})
	}
	return &assetcategorypb.ListAssetCategoriesWithPolicyRollupResponse{
		Data:    items,
		Success: true,
	}
}
