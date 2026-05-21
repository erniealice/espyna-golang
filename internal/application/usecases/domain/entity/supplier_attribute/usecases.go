package supplier_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// UseCases contains all supplier attribute-related use cases
type UseCases struct {
	CreateSupplierAttribute          *CreateSupplierAttributeUseCase
	ReadSupplierAttribute            *ReadSupplierAttributeUseCase
	UpdateSupplierAttribute          *UpdateSupplierAttributeUseCase
	DeleteSupplierAttribute          *DeleteSupplierAttributeUseCase
	ListSupplierAttributes           *ListSupplierAttributesUseCase
	GetSupplierAttributeListPageData *GetSupplierAttributeListPageDataUseCase
	GetSupplierAttributeItemPageData *GetSupplierAttributeItemPageDataUseCase
}

// SupplierAttributeRepositories groups all repository dependencies for supplier attribute use cases
type SupplierAttributeRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
	Supplier          supplierpb.SupplierDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// SupplierAttributeServices groups all business service dependencies for supplier attribute use cases
type SupplierAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of supplier attribute use cases
func NewUseCases(
	repositories SupplierAttributeRepositories,
	services SupplierAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateSupplierAttributeRepositories(repositories)
	createServices := CreateSupplierAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	readServices := ReadSupplierAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
		Supplier:          repositories.Supplier,
		Attribute:         repositories.Attribute,
	}
	updateServices := UpdateSupplierAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	deleteServices := DeleteSupplierAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListSupplierAttributesRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	listServices := ListSupplierAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetSupplierAttributeListPageDataRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	getListPageDataServices := GetSupplierAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetSupplierAttributeItemPageDataRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	getItemPageDataServices := GetSupplierAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateSupplierAttribute:          NewCreateSupplierAttributeUseCase(createRepos, createServices),
		ReadSupplierAttribute:            NewReadSupplierAttributeUseCase(readRepos, readServices),
		UpdateSupplierAttribute:          NewUpdateSupplierAttributeUseCase(updateRepos, updateServices),
		DeleteSupplierAttribute:          NewDeleteSupplierAttributeUseCase(deleteRepos, deleteServices),
		ListSupplierAttributes:           NewListSupplierAttributesUseCase(listRepos, listServices),
		GetSupplierAttributeListPageData: NewGetSupplierAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetSupplierAttributeItemPageData: NewGetSupplierAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of supplier attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer,
	supplierRepo supplierpb.SupplierDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UseCases {
	repositories := SupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
		Supplier:          supplierRepo,
		Attribute:         attributeRepo,
	}

	services := SupplierAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
