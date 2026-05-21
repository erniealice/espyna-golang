package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/fulfillment"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeFulfillment creates all fulfillment use cases from provider repositories.
func InitializeFulfillment(
	repos *domain.FulfillmentRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*fulfillment.UseCases, error) {
	return fulfillment.NewUseCases(
		fulfillment.Repositories{
			Fulfillment: repos.Fulfillment,
		},
		fulfillment.Services{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	), nil
}
