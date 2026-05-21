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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	SetAssetActive       *SetAssetActiveUseCase
}

// NewUseCases creates a new collection of asset use cases
func NewUseCases(
	repositories AssetRepositories,
	services AssetServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAssetRepositories(repositories)
	createServices := CreateAssetServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAssetRepositories(repositories)
	readServices := ReadAssetServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateAssetRepositories(repositories)
	updateServices := UpdateAssetServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteAssetRepositories(repositories)
	deleteServices := DeleteAssetServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListAssetsRepositories(repositories)
	listServices := ListAssetsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetAssetListPageDataRepositories(repositories)
	getListPageDataServices := GetAssetListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetAssetItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAssetItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	setAssetActiveRepos := SetAssetActiveRepositories(repositories)
	setAssetActiveServices := SetAssetActiveServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateAsset:          NewCreateAssetUseCase(createRepos, createServices),
		ReadAsset:            NewReadAssetUseCase(readRepos, readServices),
		UpdateAsset:          NewUpdateAssetUseCase(updateRepos, updateServices),
		DeleteAsset:          NewDeleteAssetUseCase(deleteRepos, deleteServices),
		ListAssets:           NewListAssetsUseCase(listRepos, listServices),
		GetAssetListPageData: NewGetAssetListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAssetItemPageData: NewGetAssetItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		SetAssetActive:       NewSetAssetActiveUseCase(setAssetActiveRepos, setAssetActiveServices),
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
