package supplier_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadSupplierCategoryRepositories(repositories)
	readServices := ReadSupplierCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateSupplierCategoryRepositories(repositories)
	updateServices := UpdateSupplierCategoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	deleteRepos := DeleteSupplierCategoryRepositories(repositories)
	deleteServices := DeleteSupplierCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListSupplierCategoriesRepositories(repositories)
	listServices := ListSupplierCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetSupplierCategoryListPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getListPageDataServices := GetSupplierCategoryListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetSupplierCategoryItemPageDataRepositories{
		SupplierCategory: repositories.SupplierCategory,
	}
	getItemPageDataServices := GetSupplierCategoryItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
