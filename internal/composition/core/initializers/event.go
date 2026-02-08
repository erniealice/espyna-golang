package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeEvent creates all event use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEvent(
	repos *domain.EventRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*event.EventUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return event.NewEventUseCases(
		repos.Event,
		repos.EventAttribute,
		repos.EventClient,
		repos.Client,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
