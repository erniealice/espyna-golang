package eventattendee

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// UseCases contains all event attendee-related use cases
type UseCases struct {
	CreateEventAttendee          *CreateEventAttendeeUseCase
	ReadEventAttendee            *ReadEventAttendeeUseCase
	UpdateEventAttendee          *UpdateEventAttendeeUseCase
	DeleteEventAttendee          *DeleteEventAttendeeUseCase
	ListEventAttendees           *ListEventAttendeesUseCase
	GetEventAttendeeListPageData *GetEventAttendeeListPageDataUseCase
	GetEventAttendeeItemPageData *GetEventAttendeeItemPageDataUseCase
}

// EventAttendeeRepositories groups all repository dependencies for event attendee use cases
type EventAttendeeRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// EventAttendeeServices groups all business service dependencies for event attendee use cases
type EventAttendeeServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event attendee use cases
func NewUseCases(
	repositories EventAttendeeRepositories,
	services EventAttendeeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	createServices := CreateEventAttendeeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	readServices := ReadEventAttendeeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	updateServices := UpdateEventAttendeeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	deleteServices := DeleteEventAttendeeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventAttendeesRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	listServices := ListEventAttendeesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetEventAttendeeListPageDataRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	getListPageDataServices := GetEventAttendeeListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetEventAttendeeItemPageDataRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	getItemPageDataServices := GetEventAttendeeItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateEventAttendee:          NewCreateEventAttendeeUseCase(createRepos, createServices),
		ReadEventAttendee:            NewReadEventAttendeeUseCase(readRepos, readServices),
		UpdateEventAttendee:          NewUpdateEventAttendeeUseCase(updateRepos, updateServices),
		DeleteEventAttendee:          NewDeleteEventAttendeeUseCase(deleteRepos, deleteServices),
		ListEventAttendees:           NewListEventAttendeesUseCase(listRepos, listServices),
		GetEventAttendeeListPageData: NewGetEventAttendeeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetEventAttendeeItemPageData: NewGetEventAttendeeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event attendee use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         eventRepo,
	}

	services := EventAttendeeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
