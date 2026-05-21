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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadSupplierCategoryRepositories(repositories)
	readServices := ReadSupplierCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateSupplierCategoryRepositories(repositories)
	updateServices := UpdateSupplierCategoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	deleteRepos := DeleteSupplierCategoryRepositories(repositories)
	deleteServices := DeleteSupplierCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListSupplierCategoriesRepositories(repositories)
	listServices := ListSupplierCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetSupplierCategoryListPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getListPageDataServices := GetSupplierCategoryListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetSupplierCategoryItemPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getItemPageDataServices := GetSupplierCategoryItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
