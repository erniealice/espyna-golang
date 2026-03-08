package asset

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// AssetRepositories groups all repository dependencies for asset use cases
type AssetRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// AssetServices groups all business service dependencies for asset use cases
type AssetServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all asset-related use cases
type UseCases struct {
	CreateAsset          *CreateAssetUseCase
	ReadAsset            *ReadAssetUseCase
	UpdateAsset          *UpdateAssetUseCase
	DeleteAsset          *DeleteAssetUseCase
	ListAssets           *ListAssetsUseCase
	GetAssetListPageData *GetAssetListPageDataUseCase
	GetAssetItemPageData *GetAssetItemPageDataUseCase
}

// NewUseCases creates a new collection of asset use cases
func NewUseCases(
	repositories AssetRepositories,
	services AssetServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAssetRepositories(repositories)
	createServices := CreateAssetServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadAssetRepositories(repositories)
	readServices := ReadAssetServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateAssetRepositories(repositories)
	updateServices := UpdateAssetServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteAssetRepositories(repositories)
	deleteServices := DeleteAssetServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListAssetsRepositories(repositories)
	listServices := ListAssetsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetAssetListPageDataRepositories(repositories)
	getListPageDataServices := GetAssetListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetAssetItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAssetItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateAsset:          NewCreateAssetUseCase(createRepos, createServices),
		ReadAsset:            NewReadAssetUseCase(readRepos, readServices),
		UpdateAsset:          NewUpdateAssetUseCase(updateRepos, updateServices),
		DeleteAsset:          NewDeleteAssetUseCase(deleteRepos, deleteServices),
		ListAssets:           NewListAssetsUseCase(listRepos, listServices),
		GetAssetListPageData: NewGetAssetListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAssetItemPageData: NewGetAssetItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of asset use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(assetRepo assetpb.AssetDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := AssetRepositories{
		Asset: assetRepo,
	}

	services := AssetServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
