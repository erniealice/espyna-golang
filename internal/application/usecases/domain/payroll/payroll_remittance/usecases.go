package payrollremittance

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

// Repositories groups all repository dependencies for payroll remittance use cases.
type Repositories struct {
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// Services groups all business service dependencies for payroll remittance use cases.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all payroll remittance-related use cases.
type UseCases struct {
	CreatePayrollRemittance          *CreatePayrollRemittanceUseCase
	ListPayrollRemittances           *ListPayrollRemittancesUseCase
	GetPayrollRemittanceListPageData *GetPayrollRemittanceListPageDataUseCase
}

// NewUseCases creates a new collection of payroll remittance use cases.
func NewUseCases(
	repositories Repositories,
	services Services,
) *UseCases {
	createRepos := newCreatePayrollRemittanceRepositories(repositories)
	createServices := CreatePayrollRemittanceServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := newListPayrollRemittancesRepositories(repositories)
	listServices := ListPayrollRemittancesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := newGetPayrollRemittanceListPageDataRepositories(repositories)
	getListPageDataServices := GetPayrollRemittanceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePayrollRemittance:          NewCreatePayrollRemittanceUseCase(createRepos, createServices),
		ListPayrollRemittances:           NewListPayrollRemittancesUseCase(listRepos, listServices),
		GetPayrollRemittanceListPageData: NewGetPayrollRemittanceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
