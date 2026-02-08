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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of invoice attribute use cases
func NewUseCases(
	repositories InvoiceAttributeRepositories,
	services InvoiceAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateInvoiceAttributeRepositories(repositories)
	createServices := CreateInvoiceAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	readServices := ReadInvoiceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
		Invoice:          repositories.Invoice,
		Attribute:        repositories.Attribute,
	}
	updateServices := UpdateInvoiceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteInvoiceAttributeRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	deleteServices := DeleteInvoiceAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListInvoiceAttributesRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	listServices := ListInvoiceAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetInvoiceAttributeListPageDataRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	getListPageDataServices := GetInvoiceAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetInvoiceAttributeItemPageDataRepositories{
		InvoiceAttribute: repositories.InvoiceAttribute,
	}
	getItemPageDataServices := GetInvoiceAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
