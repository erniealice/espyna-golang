package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/event"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeEvent creates all event use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEvent(
	repos *domain.EventRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*event.EventUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return event.NewEventUseCases(
		repos.Event,
		repos.EventAttendee,
		repos.EventAttribute,
		repos.EventClient,
		repos.EventOccurrence,
		repos.EventProduct,
		repos.EventRecurrence,
		repos.EventResource,
		repos.EventTag,
		repos.EventTagAssignment,
		repos.Client,
		repos.Product,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}
