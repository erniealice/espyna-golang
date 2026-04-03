package loanpayment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// LoanPaymentRepositories groups all repository dependencies for loan payment use cases.
type LoanPaymentRepositories struct {
	LoanPayment loanpaymentpb.LoanPaymentDomainServiceServer
}

// LoanPaymentServices groups all business service dependencies for loan payment use cases.
type LoanPaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all loan payment-related use cases.
type UseCases struct {
	CreateLoanPayment *CreateLoanPaymentUseCase
	ListLoanPayments  *ListLoanPaymentsUseCase
}

// NewUseCases creates a new collection of loan payment use cases.
func NewUseCases(
	repositories LoanPaymentRepositories,
	services LoanPaymentServices,
) *UseCases {
	createRepos := CreateLoanPaymentRepositories(repositories)
	createServices := CreateLoanPaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := ListLoanPaymentsRepositories(repositories)
	listServices := ListLoanPaymentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateLoanPayment: NewCreateLoanPaymentUseCase(createRepos, createServices),
		ListLoanPayments:  NewListLoanPaymentsUseCase(listRepos, listServices),
	}
}
