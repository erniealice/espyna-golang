package payroll

import (
	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollremittanceuc "github.com/erniealice/espyna-golang/internal/application/usecases/payroll/payroll_remittance"
	payrollrunuc "github.com/erniealice/espyna-golang/internal/application/usecases/payroll/payroll_run"

	// Protobuf domain services for payroll repositories
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// PayrollRepositories contains all payroll domain repositories
type PayrollRepositories struct {
	PayrollRun        payrollrunpb.PayrollRunDomainServiceServer
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// PayrollUseCases contains all payroll-related use cases
type PayrollUseCases struct {
	PayrollRun        *payrollrunuc.UseCases
	PayrollRemittance *payrollremittanceuc.UseCases
}

// NewUseCases creates all payroll use cases with proper constructor injection
func NewUseCases(
	repos PayrollRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *PayrollUseCases {
	runServices := payrollrunuc.Services{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}
	runRepos := payrollrunuc.Repositories{
		PayrollRun: repos.PayrollRun,
	}

	remittanceServices := payrollremittanceuc.Services{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}
	remittanceRepos := payrollremittanceuc.Repositories{
		PayrollRemittance: repos.PayrollRemittance,
	}

	return &PayrollUseCases{
		PayrollRun:        payrollrunuc.NewUseCases(runRepos, runServices),
		PayrollRemittance: payrollremittanceuc.NewUseCases(remittanceRepos, remittanceServices),
	}
}
