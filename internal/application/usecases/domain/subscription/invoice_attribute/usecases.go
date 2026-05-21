package invoice_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
)

// UseCases contains all invoice attribute-related use cases
type UseCases struct {
	CreateInvoiceAttribute          *CreateInvoiceAttributeUseCase
	ReadInvoiceAttribute            *ReadInvoiceAttributeUseCase
	UpdateInvoiceAttribute          *UpdateInvoiceAttributeUseCase
	DeleteInvoiceAttribute          *DeleteInvoiceAttributeUseCase
	ListInvoiceAttributes           *ListInvoiceAttributesUseCase
	GetInvoiceAttributeListPageData *GetInvoiceAttributeListPageDataUseCase
	GetInvoiceAttributeItemPageData *GetInvoiceAttributeItemPageDataUseCase
}

// InvoiceAttributeRepositories groups all repository dependencies for invoice attribute use cases
type InvoiceAttributeRepositories struct {
	InvoiceAttribute invoiceattributepb.InvoiceAttributeDomainServiceServer // Primary entity repository
	Invoice          invoicepb.InvoiceDomainServiceServer                   // Entity reference validation
	Attribute        attributepb.AttributeDomainServiceServer               // Entity reference validation
}

// InvoiceAttributeServices groups all business service dependencies for invoice attribute use cases
type InvoiceAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of invoice attribute use cases
func NewUseCases(
	repositories InvoiceAttributeRepositories,
	services InvoiceAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateInvoiceAttributeRepositories(repositories)
	createServices := CreateInvoiceAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	readServices := ReadInvoiceAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
		Invoice:          repositories.Invoice,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateInvoiceAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	deleteServices := DeleteInvoiceAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInvoiceAttributesRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	listServices := ListInvoiceAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetInvoiceAttributeListPageDataRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	getListPageDataServices := GetInvoiceAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetInvoiceAttributeItemPageDataRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	getItemPageDataServices := GetInvoiceAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInvoiceAttribute:          NewCreateInvoiceAttributeUseCase(createRepos, createServices),
		ReadInvoiceAttribute:            NewReadInvoiceAttributeUseCase(readRepos, readServices),
		UpdateInvoiceAttribute:          NewUpdateInvoiceAttributeUseCase(updateRepos, updateServices),
		DeleteInvoiceAttribute:          NewDeleteInvoiceAttributeUseCase(deleteRepos, deleteServices),
		ListInvoiceAttributes:           NewListInvoiceAttributesUseCase(listRepos, listServices),
		GetInvoiceAttributeListPageData: NewGetInvoiceAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetInvoiceAttributeItemPageData: NewGetInvoiceAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
