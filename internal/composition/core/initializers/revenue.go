package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeRevenue creates all revenue use cases from provider repositories
func InitializeRevenue(
	repos *domain.RevenueRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*revenue.RevenueUseCases, error) {
	return revenue.NewUseCases(
		revenue.RevenueRepositories{
			Revenue:          repos.Revenue,
			RevenueLineItem:  repos.RevenueLineItem,
			RevenueCategory:  repos.RevenueCategory,
			RevenueAttribute: repos.RevenueAttribute,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
