package invoice

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// InvoiceRepositories groups all repository dependencies for invoice use cases
type InvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// InvoiceServices groups all business service dependencies for invoice use cases
type InvoiceServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	IDGenerator ports.IDGenerator // Only for CreateInvoice
}

// UseCases contains all invoice-related use cases
type UseCases struct {
	CreateInvoice          *CreateInvoiceUseCase
	ReadInvoice            *ReadInvoiceUseCase
	UpdateInvoice          *UpdateInvoiceUseCase
	DeleteInvoice          *DeleteInvoiceUseCase
	ListInvoices           *ListInvoicesUseCase
	GetInvoiceListPageData *GetInvoiceListPageDataUseCase
	GetInvoiceItemPageData *GetInvoiceItemPageDataUseCase
}

// NewUseCases creates a new collection of invoice use cases
func NewUseCases(
	repositories InvoiceRepositories,
	services InvoiceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateInvoiceRepositories(repositories)
	createServices := CreateInvoiceServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadInvoiceRepositories(repositories)
	readServices := ReadInvoiceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateInvoiceRepositories(repositories)
	updateServices := UpdateInvoiceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteInvoiceRepositories(repositories)
	deleteServices := DeleteInvoiceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListInvoicesRepositories(repositories)
	listServices := ListInvoicesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetInvoiceListPageDataRepositories{
		Invoice: repositories.Invoice,
	}
	listPageDataServices := GetInvoiceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetInvoiceItemPageDataRepositories{
		Invoice: repositories.Invoice,
	}
	itemPageDataServices := GetInvoiceItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateInvoice:          NewCreateInvoiceUseCase(createRepos, createServices),
		ReadInvoice:            NewReadInvoiceUseCase(readRepos, readServices),
		UpdateInvoice:          NewUpdateInvoiceUseCase(updateRepos, updateServices),
		DeleteInvoice:          NewDeleteInvoiceUseCase(deleteRepos, deleteServices),
		ListInvoices:           NewListInvoicesUseCase(listRepos, listServices),
		GetInvoiceListPageData: NewGetInvoiceListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetInvoiceItemPageData: NewGetInvoiceItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
