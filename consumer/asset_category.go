package consumer

import (
	"context"

	asset_category_usecase "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_category"
)

// AssetCategoryWithRollup re-exports the use case type so view packages can
// reference it via the consumer package without importing espyna internals.
type AssetCategoryWithRollup = asset_category_usecase.AssetCategoryWithRollup

// ListAssetCategoriesWithPolicyRollup returns all active asset categories enriched
// with per-category IN_SERVICE asset counts and deviating-asset counts.
//
// workspace_id is derived from the request context by the workspace-aware
// adapter layer (no explicit workspace_id parameter required).
//
// Nil-safe: returns an empty slice without error when use cases are unavailable.
func ListAssetCategoriesWithPolicyRollup(
	useCases *UseCases,
	ctx context.Context,
) ([]AssetCategoryWithRollup, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.AssetCategory == nil {
		return []AssetCategoryWithRollup{}, nil
	}
	uc := useCases.Asset.AssetCategory.ListAssetCategoriesWithPolicyRollup
	if uc == nil {
		return []AssetCategoryWithRollup{}, nil
	}
	return uc.Execute(ctx)
}
