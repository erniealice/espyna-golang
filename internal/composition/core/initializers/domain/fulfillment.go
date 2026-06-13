package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/fulfillment"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeFulfillment creates all fulfillment use cases from provider repositories.
func InitializeFulfillment(
	repos *domain.FulfillmentRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*fulfillment.UseCases, error) {
	return fulfillment.NewUseCases(
		fulfillment.Repositories{
			Fulfillment: repos.Fulfillment,
		},
		fulfillment.Services{
			Authorizer:      authSvc,
			Transactor:      txSvc,
			Translator:      i18nSvc,
			IDGenerator:     idSvc,
			ActionGatekeeper: actionGate,
		},
	), nil
}
