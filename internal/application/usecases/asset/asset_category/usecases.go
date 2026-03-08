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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all asset category-related use cases
type UseCases struct {
	CreateAssetCategory          *CreateAssetCategoryUseCase
	ReadAssetCategory            *ReadAssetCategoryUseCase
	UpdateAssetCategory          *UpdateAssetCategoryUseCase
	DeleteAssetCategory          *DeleteAssetCategoryUseCase
	ListAssetCategories          *ListAssetCategoriesUseCase
	GetAssetCategoryListPageData *GetAssetCategoryListPageDataUseCase
	GetAssetCategoryItemPageData *GetAssetCategoryItemPageDataUseCase
}

// NewUseCases creates a new collection of asset category use cases
func NewUseCases(
	repositories AssetCategoryRepositories,
	services AssetCategoryServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAssetCategoryRepositories(repositories)
	createServices := CreateAssetCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadAssetCategoryRepositories(repositories)
	readServices := ReadAssetCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateAssetCategoryRepositories(repositories)
	updateServices := UpdateAssetCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteAssetCategoryRepositories(repositories)
	deleteServices := DeleteAssetCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListAssetCategoriesRepositories(repositories)
	listServices := ListAssetCategoriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetAssetCategoryListPageDataRepositories(repositories)
	getListPageDataServices := GetAssetCategoryListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetAssetCategoryItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAssetCategoryItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateAssetCategory:          NewCreateAssetCategoryUseCase(createRepos, createServices),
		ReadAssetCategory:            NewReadAssetCategoryUseCase(readRepos, readServices),
		UpdateAssetCategory:          NewUpdateAssetCategoryUseCase(updateRepos, updateServices),
		DeleteAssetCategory:          NewDeleteAssetCategoryUseCase(deleteRepos, deleteServices),
		ListAssetCategories:          NewListAssetCategoriesUseCase(listRepos, listServices),
		GetAssetCategoryListPageData: NewGetAssetCategoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAssetCategoryItemPageData: NewGetAssetCategoryItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
