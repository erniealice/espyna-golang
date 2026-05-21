package supplier

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	readServices := ReadSupplierServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	updateServices := UpdateSupplierServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteSupplierRepositories{
		Supplier: repositories.Supplier,
	}
	deleteServices := DeleteSupplierServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListSuppliersRepositories{
		Supplier: repositories.Supplier,
	}
	listServices := ListSuppliersServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetSupplierListPageDataRepositories{
		Supplier: repositories.Supplier,
	}
	getListPageDataServices := GetSupplierListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetSupplierItemPageDataRepositories{
		Supplier: repositories.Supplier,
	}
	getItemPageDataServices := GetSupplierItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
