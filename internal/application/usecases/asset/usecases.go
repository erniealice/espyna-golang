package asset

import (
	assetUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset"
	assetCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_category"
)

// AssetUseCases contains all asset domain-level use cases
type AssetUseCases struct {
	Asset         *assetUseCases.UseCases
	AssetCategory *assetCategoryUseCases.UseCases
}

// NewAssetUseCases creates a new AssetUseCases bundle from the two sub-domain use case sets.
func NewAssetUseCases(asset *assetUseCases.UseCases, assetCategory *assetCategoryUseCases.UseCases) *AssetUseCases {
	return &AssetUseCases{
		Asset:         asset,
		AssetCategory: assetCategory,
	}
}
