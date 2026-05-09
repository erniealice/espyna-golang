package asset

import (
	assetUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset"
	assetCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_category"
	assetRevaluationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_revaluation"
	depreciationRunUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/depreciation_run"
)

// AssetUseCases contains all asset domain-level use cases
type AssetUseCases struct {
	Asset            *assetUseCases.UseCases
	AssetCategory    *assetCategoryUseCases.UseCases
	DepreciationRun  *depreciationRunUseCases.UseCases
	AssetRevaluation *assetRevaluationUseCases.UseCases
}

// NewAssetUseCases creates a new AssetUseCases bundle from all sub-domain use case sets.
func NewAssetUseCases(
	asset *assetUseCases.UseCases,
	assetCategory *assetCategoryUseCases.UseCases,
	depreciationRun *depreciationRunUseCases.UseCases,
	assetRevaluation *assetRevaluationUseCases.UseCases,
) *AssetUseCases {
	return &AssetUseCases{
		Asset:            asset,
		AssetCategory:    assetCategory,
		DepreciationRun:  depreciationRun,
		AssetRevaluation: assetRevaluation,
	}
}
