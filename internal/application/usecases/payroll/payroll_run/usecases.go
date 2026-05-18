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
//
// 20260518-hexagonal-strict-adherence Phase 3 — Calculate + GeneratePayCycles
// (formerly flat fields on PayrollUseCases) now nest here. The parent payroll
// aggregator post-assigns them after constructing the Orchestrator (which is
// required by both wrappers). nil-safe.
type UseCases struct {
	CreatePayrollRun          *CreatePayrollRunUseCase
	ReadPayrollRun            *ReadPayrollRunUseCase
	ListPayrollRuns           *ListPayrollRunsUseCase
	GetPayrollRunListPageData *GetPayrollRunListPageDataUseCase

	// Orchestrator-backed run flows (Phase 3 F6 closure). Populated by the
	// parent payroll aggregator after the Orchestrator is constructed.
	Calculate        *CalculatePayrollRunUseCase
	GeneratePayCycles *GeneratePayCyclesUseCase
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
