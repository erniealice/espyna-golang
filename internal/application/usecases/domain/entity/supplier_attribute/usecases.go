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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of supplier attribute use cases
func NewUseCases(
	repositories SupplierAttributeRepositories,
	services SupplierAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateSupplierAttributeRepositories(repositories)
	createServices := CreateSupplierAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	readServices := ReadSupplierAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
		Supplier:          repositories.Supplier,
		Attribute:         repositories.Attribute,
	}
	updateServices := UpdateSupplierAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteSupplierAttributeRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	deleteServices := DeleteSupplierAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListSupplierAttributesRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	listServices := ListSupplierAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetSupplierAttributeListPageDataRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	getListPageDataServices := GetSupplierAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetSupplierAttributeItemPageDataRepositories{
		SupplierAttribute: repositories.SupplierAttribute,
	}
	getItemPageDataServices := GetSupplierAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
	authorizationService ports.Authorizer,
) *UseCases {
	repositories := SupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
		Supplier:          supplierRepo,
		Attribute:         attributeRepo,
	}

	services := SupplierAttributeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
