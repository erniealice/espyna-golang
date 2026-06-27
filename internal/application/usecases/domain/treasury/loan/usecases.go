package loan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

// LoanRepositories groups all repository dependencies for loan use cases.
type LoanRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// LoanServices groups all business service dependencies for loan use cases.
type LoanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all loan-related use cases.
type UseCases struct {
	CreateLoan          *CreateLoanUseCase
	ReadLoan            *ReadLoanUseCase
	ListLoans           *ListLoansUseCase
	GetLoanListPageData *GetLoanListPageDataUseCase
}

// NewUseCases creates a new collection of loan use cases.
func NewUseCases(
	repositories LoanRepositories,
	services LoanServices,
) *UseCases {
	createRepos := CreateLoanRepositories(repositories)
	createServices := CreateLoanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLoanRepositories(repositories)
	readServices := ReadLoanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListLoansRepositories(repositories)
	listServices := ListLoansServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetLoanListPageDataRepositories(repositories)
	getListPageDataServices := GetLoanListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateLoan:          NewCreateLoanUseCase(createRepos, createServices),
		ReadLoan:            NewReadLoanUseCase(readRepos, readServices),
		ListLoans:           NewListLoansUseCase(listRepos, listServices),
		GetLoanListPageData: NewGetLoanListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
