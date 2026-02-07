package event

import (
	"leapfor.xyz/espyna/internal/application/ports"

	// Event use cases
	eventUseCases "leapfor.xyz/espyna/internal/application/usecases/event/event"
	eventAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/event/event_attribute"
	eventClientUseCases "leapfor.xyz/espyna/internal/application/usecases/event/event_client"

	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventattributepb "leapfor.xyz/esqyma/golang/v1/domain/event/event_attribute"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// EventUseCases contains all event-related use cases
type EventUseCases struct {
	Event          *eventUseCases.UseCases
	EventAttribute *eventAttributeUseCases.UseCases
	EventClient    *eventClientUseCases.UseCases
}

// NewEventUseCases creates a new collection of event use cases
func NewEventUseCases(
	eventRepo eventpb.EventDomainServiceServer,
	eventAttributeRepo eventattributepb.EventAttributeDomainServiceServer,
	eventClientRepo eventclientpb.EventClientDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	authorizationService ports.AuthorizationService,
	transactionService ports.TransactionService,
	translationService ports.TranslationService,
	idService ports.IDService,
) *EventUseCases {
	eventRepositories := eventUseCases.EventRepositories{
		Event: eventRepo,
	}
	eventServices := eventUseCases.EventServices{
		AuthorizationService: authorizationService,
		TransactionService:   transactionService,
		TranslationService:   translationService,
		IDService:            idService,
	}

	eventAttributeRepositories := eventAttributeUseCases.EventAttributeRepositories{
		EventAttribute: eventAttributeRepo,
		Event:          eventRepo,
	}
	eventAttributeServices := eventAttributeUseCases.EventAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   transactionService,
		TranslationService:   translationService,
		IDService:            idService,
	}

	eventClientRepositories := eventClientUseCases.EventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}
	eventClientServices := eventClientUseCases.EventClientServices{
		AuthorizationService: authorizationService,
		TransactionService:   transactionService,
		TranslationService:   translationService,
		IDService:            idService,
	}

	return &EventUseCases{
		Event:          eventUseCases.NewUseCases(eventRepositories, eventServices, transactionService),
		EventAttribute: eventAttributeUseCases.NewUseCases(eventAttributeRepositories, eventAttributeServices),
		EventClient:    eventClientUseCases.NewUseCases(eventClientRepositories, eventClientServices),
	}
}
