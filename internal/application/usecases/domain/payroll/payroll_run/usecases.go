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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	Calculate         *CalculatePayrollRunUseCase
	GeneratePayCycles *GeneratePayCyclesUseCase
}

// NewUseCases creates a new collection of payroll run use cases.
func NewUseCases(
	repositories Repositories,
	services Services,
) *UseCases {
	createRepos := newCreatePayrollRunRepositories(repositories)
	createServices := CreatePayrollRunServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := newReadPayrollRunRepositories(repositories)
	readServices := ReadPayrollRunServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := newListPayrollRunsRepositories(repositories)
	listServices := ListPayrollRunsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := newGetPayrollRunListPageDataRepositories(repositories)
	getListPageDataServices := GetPayrollRunListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePayrollRun:          NewCreatePayrollRunUseCase(createRepos, createServices),
		ReadPayrollRun:            NewReadPayrollRunUseCase(readRepos, readServices),
		ListPayrollRuns:           NewListPayrollRunsUseCase(listRepos, listServices),
		GetPayrollRunListPageData: NewGetPayrollRunListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
