package invoice

import (
	"leapfor.xyz/espyna/internal/application/ports"
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
)

// InvoiceRepositories groups all repository dependencies for invoice use cases
type InvoiceRepositories struct {
	Invoice invoicepb.InvoiceDomainServiceServer // Primary entity repository
}

// InvoiceServices groups all business service dependencies for invoice use cases
type InvoiceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreateInvoice
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadInvoiceRepositories(repositories)
	readServices := ReadInvoiceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateInvoiceRepositories(repositories)
	updateServices := UpdateInvoiceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteInvoiceRepositories(repositories)
	deleteServices := DeleteInvoiceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListInvoicesRepositories(repositories)
	listServices := ListInvoicesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetInvoiceListPageDataRepositories{
		Invoice: repositories.Invoice,
	}
	listPageDataServices := GetInvoiceListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetInvoiceItemPageDataRepositories{
		Invoice: repositories.Invoice,
	}
	itemPageDataServices := GetInvoiceItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
