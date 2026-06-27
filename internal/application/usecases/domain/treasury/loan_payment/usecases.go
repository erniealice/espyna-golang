package loanpayment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// LoanPaymentRepositories groups all repository dependencies for loan payment use cases.
type LoanPaymentRepositories struct {
	LoanPayment loanpaymentpb.LoanPaymentDomainServiceServer
}

// LoanPaymentServices groups all business service dependencies for loan payment use cases.
type LoanPaymentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListLoanPaymentsRepositories(repositories)
	listServices := ListLoanPaymentsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateLoanPayment: NewCreateLoanPaymentUseCase(createRepos, createServices),
		ListLoanPayments:  NewListLoanPaymentsUseCase(listRepos, listServices),
	}
}
