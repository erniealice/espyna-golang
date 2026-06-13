package supplier

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// SupplierRepositories groups all repository dependencies for supplier use cases
type SupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
	User     userpb.UserDomainServiceServer         // User repository for embedded user data
}

// SupplierServices groups all business service dependencies for supplier use cases
type SupplierServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all supplier-related use cases
type UseCases struct {
	CreateSupplier          *CreateSupplierUseCase
	ReadSupplier            *ReadSupplierUseCase
	UpdateSupplier          *UpdateSupplierUseCase
	DeleteSupplier          *DeleteSupplierUseCase
	ListSuppliers           *ListSuppliersUseCase
	GetSupplierListPageData *GetSupplierListPageDataUseCase
	GetSupplierItemPageData *GetSupplierItemPageDataUseCase
}

// NewUseCases creates a new collection of supplier use cases
func NewUseCases(
	repositories SupplierRepositories,
	services SupplierServices,
) *UseCases {
	createRepos := CreateSupplierRepositories{
		Supplier: repositories.Supplier,
		User:     repositories.User,
	}
	createServices := CreateSupplierServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	readServices := ReadSupplierServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	updateServices := UpdateSupplierServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	deleteServices := DeleteSupplierServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListSuppliersRepositories{
		Supplier: repositories.Supplier,
	}
	listServices := ListSuppliersServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetSupplierListPageDataRepositories{
		Supplier: repositories.Supplier,
	}
	getListPageDataServices := GetSupplierListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetSupplierItemPageDataRepositories{
		Supplier: repositories.Supplier,
	}
	getItemPageDataServices := GetSupplierItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateSupplier:          NewCreateSupplierUseCase(createRepos, createServices),
		ReadSupplier:            NewReadSupplierUseCase(readRepos, readServices),
		UpdateSupplier:          NewUpdateSupplierUseCase(updateRepos, updateServices),
		DeleteSupplier:          NewDeleteSupplierUseCase(deleteRepos, deleteServices),
		ListSuppliers:           NewListSuppliersUseCase(listRepos, listServices),
		GetSupplierListPageData: NewGetSupplierListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetSupplierItemPageData: NewGetSupplierItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of supplier use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *UseCases {
	repositories := SupplierRepositories{
		Supplier: supplierRepo,
	}

	services := SupplierServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
