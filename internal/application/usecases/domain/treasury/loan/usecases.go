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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadLoanRepositories(repositories)
	readServices := ReadLoanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListLoansRepositories(repositories)
	listServices := ListLoansServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetLoanListPageDataRepositories(repositories)
	getListPageDataServices := GetLoanListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateLoan:          NewCreateLoanUseCase(createRepos, createServices),
		ReadLoan:            NewReadLoanUseCase(readRepos, readServices),
		ListLoans:           NewListLoansUseCase(listRepos, listServices),
		GetLoanListPageData: NewGetLoanListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
