package eventattendee

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	readServices := ReadEventAttendeeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	updateServices := UpdateEventAttendeeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventAttendeeRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	deleteServices := DeleteEventAttendeeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventAttendeesRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	listServices := ListEventAttendeesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEventAttendeeListPageDataRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	getListPageDataServices := GetEventAttendeeListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetEventAttendeeItemPageDataRepositories{
		EventAttendee: repositories.EventAttendee,
		Event:         repositories.Event,
	}
	getItemPageDataServices := GetEventAttendeeItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         eventRepo,
	}

	services := EventAttendeeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
