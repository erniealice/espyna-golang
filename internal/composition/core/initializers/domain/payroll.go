package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payroll"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializePayroll creates all payroll use cases from provider repositories.
// entityRepos and expenditureRepos are required for the orchestrator's
// cross-domain reads (employees, employment contracts) and writes (Expenditure
// payslips); pass nil to fall back to a degraded mode where the calculate /
// generate-cycles use cases return an error on Execute but the rest still works.
func InitializePayroll(
	repos *domain.PayrollRepositories,
	entityRepos *domain.EntityRepositories,
	expenditureRepos *domain.ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*payroll.PayrollUseCases, error) {
	cross := payroll.CrossDomainRepositories{}
	if entityRepos != nil {
		cross.Workspace = entityRepos.Workspace
		cross.Supplier = entityRepos.Supplier
	}
	if expenditureRepos != nil {
		cross.SupplierContract = expenditureRepos.SupplierContract
		cross.SupplierContractLine = expenditureRepos.SupplierContractLine
		cross.Expenditure = expenditureRepos.Expenditure
		cross.ExpenditureLineItem = expenditureRepos.ExpenditureLineItem
	}

	return payroll.NewUseCases(
		payroll.PayrollRepositories{
			PayrollRun:        repos.PayrollRun,
			PayrollRemittance: repos.PayrollRemittance,
			PayCycle:          repos.PayCycle,
			RateTable:         repos.RateTable,
			RateBand:          repos.RateBand,
			LeaveBalance:      repos.LeaveBalance,
		},
		cross,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
