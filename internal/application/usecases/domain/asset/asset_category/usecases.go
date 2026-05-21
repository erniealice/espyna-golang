package asset_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
)

// AssetCategoryRepositories groups all repository dependencies for asset category use cases
type AssetCategoryRepositories struct {
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer // Primary entity repository
}

// AssetCategoryServices groups all business service dependencies for asset category use cases
type AssetCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all asset category-related use cases
type UseCases struct {
	CreateAssetCategory                 *CreateAssetCategoryUseCase
	ReadAssetCategory                   *ReadAssetCategoryUseCase
	UpdateAssetCategory                 *UpdateAssetCategoryUseCase
	DeleteAssetCategory                 *DeleteAssetCategoryUseCase
	ListAssetCategories                 *ListAssetCategoriesUseCase
	GetAssetCategoryListPageData        *GetAssetCategoryListPageDataUseCase
	GetAssetCategoryItemPageData        *GetAssetCategoryItemPageDataUseCase
	ListAssetCategoriesWithPolicyRollup *ListAssetCategoriesWithPolicyRollupUseCase
}

// NewUseCases creates a new collection of asset category use cases
func NewUseCases(
	repositories AssetCategoryRepositories,
	services AssetCategoryServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAssetCategoryRepositories(repositories)
	createServices := CreateAssetCategoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAssetCategoryRepositories(repositories)
	readServices := ReadAssetCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateAssetCategoryRepositories(repositories)
	updateServices := UpdateAssetCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteAssetCategoryRepositories(repositories)
	deleteServices := DeleteAssetCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListAssetCategoriesRepositories(repositories)
	listServices := ListAssetCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetAssetCategoryListPageDataRepositories(repositories)
	getListPageDataServices := GetAssetCategoryListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetAssetCategoryItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAssetCategoryItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	rollupRepos := ListAssetCategoriesWithPolicyRollupRepositories{
		AssetCategory: repositories.AssetCategory,
	}
	rollupServices := ListAssetCategoriesWithPolicyRollupServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateAssetCategory:                 NewCreateAssetCategoryUseCase(createRepos, createServices),
		ReadAssetCategory:                   NewReadAssetCategoryUseCase(readRepos, readServices),
		UpdateAssetCategory:                 NewUpdateAssetCategoryUseCase(updateRepos, updateServices),
		DeleteAssetCategory:                 NewDeleteAssetCategoryUseCase(deleteRepos, deleteServices),
		ListAssetCategories:                 NewListAssetCategoriesUseCase(listRepos, listServices),
		GetAssetCategoryListPageData:        NewGetAssetCategoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAssetCategoryItemPageData:        NewGetAssetCategoryItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		ListAssetCategoriesWithPolicyRollup: NewListAssetCategoriesWithPolicyRollupUseCase(rollupRepos, rollupServices),
	}
}

// NewUseCasesUngrouped creates a new collection of asset category use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(assetCategoryRepo assetcategorypb.AssetCategoryDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := AssetCategoryRepositories{
		AssetCategory: assetCategoryRepo,
	}

	services := AssetCategoryServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
