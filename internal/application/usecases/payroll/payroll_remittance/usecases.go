package payrollremittance

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

// Repositories groups all repository dependencies for payroll remittance use cases.
type Repositories struct {
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// Services groups all business service dependencies for payroll remittance use cases.
type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := newListPayrollRemittancesRepositories(repositories)
	listServices := ListPayrollRemittancesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := newGetPayrollRemittanceListPageDataRepositories(repositories)
	getListPageDataServices := GetPayrollRemittanceListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePayrollRemittance:          NewCreatePayrollRemittanceUseCase(createRepos, createServices),
		ListPayrollRemittances:           NewListPayrollRemittancesUseCase(listRepos, listServices),
		GetPayrollRemittanceListPageData: NewGetPayrollRemittanceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
