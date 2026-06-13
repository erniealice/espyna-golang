package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset"
	assetUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset/asset"
	assetCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset/asset_category"
	assetRevaluationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset/asset_revaluation"
	depreciationRunUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset/depreciation_run"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeAsset creates all asset use cases from provider repositories.
// This is composition logic - it wires infrastructure (providers) to application (use cases).
func InitializeAsset(
	repos *domain.AssetRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*asset.AssetUseCases, error) {
	// Build the Asset sub-bundle
	assetSub := assetUseCases.NewUseCases(
		assetUseCases.AssetRepositories{
			Asset: repos.Asset,
		},
		assetUseCases.AssetServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
		},
	)

	// Build the AssetCategory sub-bundle
	assetCategorySub := assetCategoryUseCases.NewUseCases(
		assetCategoryUseCases.AssetCategoryRepositories{
			AssetCategory: repos.AssetCategory,
		},
		assetCategoryUseCases.AssetCategoryServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idSvc,
			ActionGatekeeper: actionGate,
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
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idSvc,
		ActionGatekeeper: actionGate,
	}
	depRunSub := depreciationRunUseCases.NewUseCases(depRunRepos, depRunSvc)

	// Build the AssetRevaluation sub-bundle
	revRepos := assetRevaluationUseCases.AssetRevaluationRepositories{
		Asset:            repos.Asset,
		AssetTransaction: repos.AssetTransaction,
		AssetRevaluation: repos.AssetRevaluation,
	}
	revSvc := assetRevaluationUseCases.AssetRevaluationServices{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idSvc,
		ActionGatekeeper: actionGate,
	}
	revSub := assetRevaluationUseCases.NewUseCases(revRepos, revSvc)

	return asset.NewAssetUseCases(assetSub, assetCategorySub, depRunSub, revSub), nil
}
