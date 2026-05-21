package event

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// EventRepositories groups all repository dependencies for event use cases
type EventRepositories struct {
	Event eventpb.EventDomainServiceServer // Primary entity repository
}

// EventServices groups all business service dependencies for event use cases
type EventServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all event-related use cases
type UseCases struct {
	CreateEvent          *CreateEventUseCase
	ReadEvent            *ReadEventUseCase
	UpdateEvent          *UpdateEventUseCase
	DeleteEvent          *DeleteEventUseCase
	ListEvents           *ListEventsUseCase
	GetEventListPageData *GetEventListPageDataUseCase
	GetEventItemPageData *GetEventItemPageDataUseCase
}

// NewUseCases creates a new collection of event use cases
func NewUseCases(
	repositories EventRepositories,
	services EventServices,
	transactionService ports.Transactor,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventRepositories(repositories)
	createServices := CreateEventServices{
		Authorizer:  services.Authorizer,
		Transactor:  transactionService,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventRepositories(repositories)
	readServices := ReadEventServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	updateRepos := UpdateEventRepositories(repositories)
	updateServices := UpdateEventServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	deleteRepos := DeleteEventRepositories(repositories)
	deleteServices := DeleteEventServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	listRepos := ListEventsRepositories(repositories)
	listServices := ListEventsServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	listPageDataRepos := GetEventListPageDataRepositories{
		Event: repositories.Event,
	}
	listPageDataServices := GetEventListPageDataServices{
		Transactor: transactionService,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetEventItemPageDataRepositories{
		Event: repositories.Event,
	}
	itemPageDataServices := GetEventItemPageDataServices{
		Transactor: transactionService,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateEvent:          NewCreateEventUseCase(createRepos, createServices),
		ReadEvent:            NewReadEventUseCase(readRepos, readServices),
		UpdateEvent:          NewUpdateEventUseCase(updateRepos, updateServices),
		DeleteEvent:          NewDeleteEventUseCase(deleteRepos, deleteServices),
		ListEvents:           NewListEventsUseCase(listRepos, listServices),
		GetEventListPageData: NewGetEventListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetEventItemPageData: NewGetEventItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(eventRepo eventpb.EventDomainServiceServer, transactionService ports.Transactor) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventRepositories{
		Event: eventRepo,
	}

	services := EventServices{
		Authorizer: nil, // Will be injected later by container
		Transactor: transactionService,
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services, transactionService)
}
