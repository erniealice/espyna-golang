package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeExpenditure creates all expenditure use cases from provider repositories
func InitializeExpenditure(
	repos *domain.ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*expenditure.ExpenditureUseCases, error) {
	return expenditure.NewUseCases(
		expenditure.ExpenditureRepositories{
			Expenditure:          repos.Expenditure,
			ExpenditureLineItem:  repos.ExpenditureLineItem,
			ExpenditureCategory:  repos.ExpenditureCategory,
			ExpenditureAttribute: repos.ExpenditureAttribute,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
