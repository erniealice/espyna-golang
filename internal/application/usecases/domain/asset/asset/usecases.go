package asset

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// AssetRepositories groups all repository dependencies for asset use cases
type AssetRepositories struct {
	Asset   assetpb.AssetDomainServiceServer // Primary entity repository
	Product productpb.ProductDomainServiceServer
}

// AssetServices groups all business service dependencies for asset use cases
type AssetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
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
	AssignProduct        *AssignProductUseCase
}

// NewUseCases creates a new collection of asset use cases
func NewUseCases(
	repositories AssetRepositories,
	services AssetServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAssetRepositories{Asset: repositories.Asset, Product: repositories.Product}
	createServices := CreateAssetServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator:      services.IDGenerator,
	}

	readRepos := ReadAssetRepositories{Asset: repositories.Asset}
	readServices := ReadAssetServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateAssetRepositories{Asset: repositories.Asset, Product: repositories.Product}
	updateServices := UpdateAssetServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteAssetRepositories{Asset: repositories.Asset}
	deleteServices := DeleteAssetServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListAssetsRepositories{Asset: repositories.Asset}
	listServices := ListAssetsServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetAssetListPageDataRepositories{Asset: repositories.Asset}
	getListPageDataServices := GetAssetListPageDataServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetAssetItemPageDataRepositories{Asset: repositories.Asset}
	getItemPageDataServices := GetAssetItemPageDataServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	setAssetActiveRepos := SetAssetActiveRepositories{Asset: repositories.Asset}
	setAssetActiveServices := SetAssetActiveServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}
	assignProductRepos := AssignProductRepositories{Asset: repositories.Asset, Product: repositories.Product}
	assignProductServices := AssignProductServices{Translator: services.Translator, ActionGatekeeper: services.ActionGatekeeper}

	return &UseCases{
		CreateAsset:          NewCreateAssetUseCase(createRepos, createServices),
		ReadAsset:            NewReadAssetUseCase(readRepos, readServices),
		UpdateAsset:          NewUpdateAssetUseCase(updateRepos, updateServices),
		DeleteAsset:          NewDeleteAssetUseCase(deleteRepos, deleteServices),
		ListAssets:           NewListAssetsUseCase(listRepos, listServices),
		GetAssetListPageData: NewGetAssetListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAssetItemPageData: NewGetAssetItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		SetAssetActive:       NewSetAssetActiveUseCase(setAssetActiveRepos, setAssetActiveServices),
		AssignProduct:        NewAssignProductUseCase(assignProductRepos, assignProductServices),
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
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
