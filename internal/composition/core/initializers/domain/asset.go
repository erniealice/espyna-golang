package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/asset"
	assetUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset"
	assetCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_category"
	assetRevaluationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_revaluation"
	depreciationRunUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/asset/depreciation_run"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeAsset creates all asset use cases from provider repositories.
// This is composition logic - it wires infrastructure (providers) to application (use cases).
func InitializeAsset(
	repos *domain.AssetRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*asset.AssetUseCases, error) {
	// Build the Asset sub-bundle
	assetSub := assetUseCases.NewUseCases(
		assetUseCases.AssetRepositories{
			Asset: repos.Asset,
		},
		assetUseCases.AssetServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	// Build the AssetCategory sub-bundle
	assetCategorySub := assetCategoryUseCases.NewUseCases(
		assetCategoryUseCases.AssetCategoryRepositories{
			AssetCategory: repos.AssetCategory,
		},
		assetCategoryUseCases.AssetCategoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	// Build the DepreciationRun sub-bundle
	depRunRepos := depreciationRunUseCases.DepreciationRunRepositories{
		Asset:                repos.Asset,
		AssetCategory:        repos.AssetCategory,
		AssetTransaction:     repos.AssetTransaction,
		DepreciationSchedule: repos.DepreciationSchedule,
		DepreciationRun:      repos.DepreciationRun,
	}
	depRunSvc := depreciationRunUseCases.DepreciationRunServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idSvc,
	}
	depRunSub := depreciationRunUseCases.NewUseCases(depRunRepos, depRunSvc)

	// Build the AssetRevaluation sub-bundle
	revRepos := assetRevaluationUseCases.AssetRevaluationRepositories{
		Asset:            repos.Asset,
		AssetTransaction: repos.AssetTransaction,
		AssetRevaluation: repos.AssetRevaluation,
	}
	revSvc := assetRevaluationUseCases.AssetRevaluationServices{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idSvc,
	}
	revSub := assetRevaluationUseCases.NewUseCases(revRepos, revSvc)

	return asset.NewAssetUseCases(assetSub, assetCategorySub, depRunSub, revSub), nil
}
