package payrollrun

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// Repositories groups all repository dependencies for payroll run use cases.
type Repositories struct {
	PayrollRun payrollrunpb.PayrollRunDomainServiceServer
}

// Services groups all business service dependencies for payroll run use cases.
type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all payroll run-related use cases.
type UseCases struct {
	CreatePayrollRun          *CreatePayrollRunUseCase
	ReadPayrollRun            *ReadPayrollRunUseCase
	ListPayrollRuns           *ListPayrollRunsUseCase
	GetPayrollRunListPageData *GetPayrollRunListPageDataUseCase
}

// NewUseCases creates a new collection of payroll run use cases.
func NewUseCases(
	repositories Repositories,
	services Services,
) *UseCases {
	createRepos := newCreatePayrollRunRepositories(repositories)
	createServices := CreatePayrollRunServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := newReadPayrollRunRepositories(repositories)
	readServices := ReadPayrollRunServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := newListPayrollRunsRepositories(repositories)
	listServices := ListPayrollRunsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := newGetPayrollRunListPageDataRepositories(repositories)
	getListPageDataServices := GetPayrollRunListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePayrollRun:          NewCreatePayrollRunUseCase(createRepos, createServices),
		ReadPayrollRun:            NewReadPayrollRunUseCase(readRepos, readServices),
		ListPayrollRuns:           NewListPayrollRunsUseCase(listRepos, listServices),
		GetPayrollRunListPageData: NewGetPayrollRunListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
