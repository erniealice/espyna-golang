package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/finance"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeFinance creates all finance use cases from provider repositories.
func InitializeFinance(
	repos *domain.FinanceRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*finance.FinanceUseCases, error) {
	return finance.NewUseCases(
		finance.FinanceRepositories{
			ForexRate: repos.ForexRate,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
