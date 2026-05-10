package asset_revaluation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
)

// AssetRevaluationRepositories groups all repository dependencies.
type AssetRevaluationRepositories struct {
	Asset            assetpb.AssetDomainServiceServer
	AssetTransaction assettxpb.AssetTransactionDomainServiceServer
	AssetRevaluation revaluation_pb.AssetRevaluationDomainServiceServer
}

// AssetRevaluationServices groups all service dependencies.
type AssetRevaluationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all asset-revaluation-related use cases.
type UseCases struct {
	RevalueAsset       *RevalueAssetUseCase
	PreviewRevaluation *PreviewRevaluationUseCase
}

// NewUseCases creates a new collection of asset-revaluation use cases.
func NewUseCases(
	repositories AssetRevaluationRepositories,
	services AssetRevaluationServices,
) *UseCases {
	revalueRepos := RevalueAssetRepositories{
		Asset:            repositories.Asset,
		AssetTransaction: repositories.AssetTransaction,
		AssetRevaluation: repositories.AssetRevaluation,
	}
	revalueServices := RevalueAssetServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	return &UseCases{
		RevalueAsset:       NewRevalueAssetUseCase(revalueRepos, revalueServices),
		PreviewRevaluation: NewPreviewRevaluationUseCase(revalueRepos, revalueServices),
	}
}
