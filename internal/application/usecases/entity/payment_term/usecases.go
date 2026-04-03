package payment_term

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// PaymentTermRepositories groups all repository dependencies for payment term use cases
type PaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// PaymentTermServices groups all business service dependencies for payment term use cases
type PaymentTermServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all payment-term-related use cases
type UseCases struct {
	CreatePaymentTerm          *CreatePaymentTermUseCase
	ReadPaymentTerm            *ReadPaymentTermUseCase
	UpdatePaymentTerm          *UpdatePaymentTermUseCase
	DeletePaymentTerm          *DeletePaymentTermUseCase
	ListPaymentTerms           *ListPaymentTermsUseCase
	GetPaymentTermListPageData *GetPaymentTermListPageDataUseCase
	GetPaymentTermItemPageData *GetPaymentTermItemPageDataUseCase
}

// NewUseCases creates a new collection of payment term use cases
func NewUseCases(
	repositories PaymentTermRepositories,
	services PaymentTermServices,
) *UseCases {
	createRepos := CreatePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	createServices := CreatePaymentTermServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	readServices := ReadPaymentTermServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	updateServices := UpdatePaymentTermServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePaymentTermRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	deleteServices := DeletePaymentTermServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPaymentTermsRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	listServices := ListPaymentTermsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPaymentTermListPageDataRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	getListPageDataServices := GetPaymentTermListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetPaymentTermItemPageDataRepositories{
		PaymentTerm: repositories.PaymentTerm,
	}
	getItemPageDataServices := GetPaymentTermItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePaymentTerm:          NewCreatePaymentTermUseCase(createRepos, createServices),
		ReadPaymentTerm:            NewReadPaymentTermUseCase(readRepos, readServices),
		UpdatePaymentTerm:          NewUpdatePaymentTermUseCase(updateRepos, updateServices),
		DeletePaymentTerm:          NewDeletePaymentTermUseCase(deleteRepos, deleteServices),
		ListPaymentTerms:           NewListPaymentTermsUseCase(listRepos, listServices),
		GetPaymentTermListPageData: NewGetPaymentTermListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPaymentTermItemPageData: NewGetPaymentTermItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of payment term use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *UseCases {
	repositories := PaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := PaymentTermServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
