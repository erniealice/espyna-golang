package loan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLoanRepositories(repositories)
	readServices := ReadLoanServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListLoansRepositories(repositories)
	listServices := ListLoansServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetLoanListPageDataRepositories(repositories)
	getListPageDataServices := GetLoanListPageDataServices{
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
