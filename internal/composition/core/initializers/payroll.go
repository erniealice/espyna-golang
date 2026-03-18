package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payroll"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializePayroll creates all payroll use cases from provider repositories
func InitializePayroll(
	repos *domain.PayrollRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*payroll.PayrollUseCases, error) {
	return payroll.NewUseCases(
		payroll.PayrollRepositories{
			PayrollRun:        repos.PayrollRun,
			PayrollRemittance: repos.PayrollRemittance,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
