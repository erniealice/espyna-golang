package event

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Event use cases
	eventUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event"
	eventAttendeeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_attendee"
	eventAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_attribute"
	eventClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_client"
	eventOccurrenceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_occurrence"
	eventProductUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_product"
	eventRecurrenceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_recurrence"
	eventResourceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_resource"
	eventTagUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_tag"
	eventTagAssignmentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/event_tag_assignment"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"

	eventAttendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
	eventOccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
	eventProductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	eventRecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	eventResourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// EventUseCases contains all event-related use cases
type EventUseCases struct {
	Event              *eventUseCases.UseCases
	EventAttendee      *eventAttendeeUseCases.UseCases
	EventAttribute     *eventAttributeUseCases.UseCases
	EventClient        *eventClientUseCases.UseCases
	EventOccurrence    *eventOccurrenceUseCases.UseCases
	EventProduct       *eventProductUseCases.UseCases
	EventRecurrence    *eventRecurrenceUseCases.UseCases
	EventResource      *eventResourceUseCases.UseCases
	EventTag           *eventTagUseCases.UseCases
	EventTagAssignment *eventTagAssignmentUseCases.UseCases
}

// NewEventUseCases creates a new collection of event use cases
func NewEventUseCases(
	eventRepo eventpb.EventDomainServiceServer,
	eventAttendeeRepo eventAttendeepb.EventAttendeeDomainServiceServer,
	eventAttributeRepo eventattributepb.EventAttributeDomainServiceServer,
	eventClientRepo eventclientpb.EventClientDomainServiceServer,
	eventOccurrenceRepo eventOccurrencepb.EventOccurrenceDomainServiceServer,
	eventProductRepo eventProductpb.EventProductDomainServiceServer,
	eventRecurrenceRepo eventRecurrencepb.EventRecurrenceDomainServiceServer,
	eventResourceRepo eventResourcepb.EventResourceDomainServiceServer,
	eventTagRepo eventtagpb.EventTagDomainServiceServer,
	eventTagAssignmentRepo eventtagassignmentpb.EventTagAssignmentDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	productRepo productpb.ProductDomainServiceServer,
	authorizationService ports.Authorizer,
	transactionService ports.Transactor,
	translationService ports.Translator,
	idService ports.IDGenerator,
) *EventUseCases {
	// Shared services for all use cases
	sharedServices := struct {
		Auth ports.Authorizer
		Tx   ports.Transactor
		I18n ports.Translator
		ID   ports.IDGenerator
	}{authorizationService, transactionService, translationService, idService}

	// Event (core)
	eventRepositories := eventUseCases.EventRepositories{Event: eventRepo}
	eventServices := eventUseCases.EventServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventAttendee
	eventAttendeeRepositories := eventAttendeeUseCases.EventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         eventRepo,
	}
	eventAttendeeServices := eventAttendeeUseCases.EventAttendeeServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventAttribute
	eventAttributeRepositories := eventAttributeUseCases.EventAttributeRepositories{
		EventAttribute: eventAttributeRepo,
		Event:          eventRepo,
	}
	eventAttributeServices := eventAttributeUseCases.EventAttributeServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventClient
	eventClientRepositories := eventClientUseCases.EventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}
	eventClientServices := eventClientUseCases.EventClientServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventOccurrence (read-only)
	eventOccurrenceRepositories := eventOccurrenceUseCases.EventOccurrenceRepositories{
		EventOccurrence: eventOccurrenceRepo,
	}
	eventOccurrenceServices := eventOccurrenceUseCases.EventOccurrenceServices{
		Authorizer: sharedServices.Auth,
		Transactor: sharedServices.Tx,
		Translator: sharedServices.I18n,
	}

	// EventProduct
	eventProductRepositories := eventProductUseCases.EventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        eventRepo,
		Product:      productRepo,
	}
	eventProductServices := eventProductUseCases.EventProductServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventRecurrence
	eventRecurrenceRepositories := eventRecurrenceUseCases.EventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}
	eventRecurrenceServices := eventRecurrenceUseCases.EventRecurrenceServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventResource
	eventResourceRepositories := eventResourceUseCases.EventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         eventRepo,
	}
	eventResourceServices := eventResourceUseCases.EventResourceServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventTag (master list, per workspace)
	eventTagRepositories := eventTagUseCases.EventTagRepositories{
		EventTag: eventTagRepo,
	}
	eventTagServices := eventTagUseCases.EventTagServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// EventTagAssignment (event ↔ tag join)
	eventTagAssignmentRepositories := eventTagAssignmentUseCases.EventTagAssignmentRepositories{
		EventTagAssignment: eventTagAssignmentRepo,
		Event:              eventRepo,
		EventTag:           eventTagRepo,
	}
	eventTagAssignmentServices := eventTagAssignmentUseCases.EventTagAssignmentServices{
		Authorizer:  sharedServices.Auth,
		Transactor:  sharedServices.Tx,
		Translator:  sharedServices.I18n,
		IDGenerator: sharedServices.ID,
	}

	// Per Wave B P1.C.7 (20260520-service-domain-migration, Q-SDM-DASHBOARD-LAYOUT)
	// the previously embedded `Dashboard *scheduledashboard.GetScheduleDashboardPageDataUseCase`
	// flat field has been absorbed into the service-driven category at
	// `internal/application/usecases/service/dashboard/schedule/` and is now
	// wired through the composition root as a Schedule entity-dashboard dep
	// on the `service/dashboard.Deps` umbrella. The entity-layer use case at
	// `internal/application/usecases/domain/event/dashboard/` is retained as the
	// algorithmic implementation; only the flat-field exposure on this
	// aggregator is removed.

	return &EventUseCases{
		Event:              eventUseCases.NewUseCases(eventRepositories, eventServices, transactionService),
		EventAttendee:      eventAttendeeUseCases.NewUseCases(eventAttendeeRepositories, eventAttendeeServices),
		EventAttribute:     eventAttributeUseCases.NewUseCases(eventAttributeRepositories, eventAttributeServices),
		EventClient:        eventClientUseCases.NewUseCases(eventClientRepositories, eventClientServices),
		EventOccurrence:    eventOccurrenceUseCases.NewUseCases(eventOccurrenceRepositories, eventOccurrenceServices),
		EventProduct:       eventProductUseCases.NewUseCases(eventProductRepositories, eventProductServices),
		EventRecurrence:    eventRecurrenceUseCases.NewUseCases(eventRecurrenceRepositories, eventRecurrenceServices, transactionService),
		EventResource:      eventResourceUseCases.NewUseCases(eventResourceRepositories, eventResourceServices),
		EventTag:           eventTagUseCases.NewUseCases(eventTagRepositories, eventTagServices),
		EventTagAssignment: eventTagAssignmentUseCases.NewUseCases(eventTagAssignmentRepositories, eventTagAssignmentServices),
	}
}
