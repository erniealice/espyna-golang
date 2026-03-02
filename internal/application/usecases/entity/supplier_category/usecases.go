package supplier_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// SupplierCategoryRepositories groups all repository dependencies for supplier_category use cases
type SupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// SupplierCategoryServices groups all business service dependencies for supplier_category use cases
type SupplierCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all supplier_category-related use cases
type UseCases struct {
	CreateSupplierCategory          *CreateSupplierCategoryUseCase
	ReadSupplierCategory            *ReadSupplierCategoryUseCase
	UpdateSupplierCategory          *UpdateSupplierCategoryUseCase
	DeleteSupplierCategory          *DeleteSupplierCategoryUseCase
	ListSupplierCategories          *ListSupplierCategoriesUseCase
	GetSupplierCategoryListPageData *GetSupplierCategoryListPageDataUseCase
	GetSupplierCategoryItemPageData *GetSupplierCategoryItemPageDataUseCase
}

// NewUseCases creates a new collection of supplier_category use cases
func NewUseCases(
	repositories SupplierCategoryRepositories,
	services SupplierCategoryServices,
) *UseCases {
	createRepos := CreateSupplierCategoryRepositories(repositories)
	createServices := CreateSupplierCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadSupplierCategoryRepositories(repositories)
	readServices := ReadSupplierCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateSupplierCategoryRepositories(repositories)
	updateServices := UpdateSupplierCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	deleteRepos := DeleteSupplierCategoryRepositories(repositories)
	deleteServices := DeleteSupplierCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListSupplierCategoriesRepositories(repositories)
	listServices := ListSupplierCategoriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetSupplierCategoryListPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getListPageDataServices := GetSupplierCategoryListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetSupplierCategoryItemPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getItemPageDataServices := GetSupplierCategoryItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateSupplierCategory:          NewCreateSupplierCategoryUseCase(createRepos, createServices),
		ReadSupplierCategory:            NewReadSupplierCategoryUseCase(readRepos, readServices),
		UpdateSupplierCategory:          NewUpdateSupplierCategoryUseCase(updateRepos, updateServices),
		DeleteSupplierCategory:          NewDeleteSupplierCategoryUseCase(deleteRepos, deleteServices),
		ListSupplierCategories:          NewListSupplierCategoriesUseCase(listRepos, listServices),
		GetSupplierCategoryListPageData: NewGetSupplierCategoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetSupplierCategoryItemPageData: NewGetSupplierCategoryItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of supplier_category use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *UseCases {
	repositories := SupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := SupplierCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
