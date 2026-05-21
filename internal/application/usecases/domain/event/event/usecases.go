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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
	transactionService ports.TransactionService,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventRepositories(repositories)
	createServices := CreateEventServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventRepositories(repositories)
	readServices := ReadEventServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventRepositories(repositories)
	updateServices := UpdateEventServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventRepositories(repositories)
	deleteServices := DeleteEventServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventsRepositories(repositories)
	listServices := ListEventsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetEventListPageDataRepositories{
		Event: repositories.Event,
	}
	listPageDataServices := GetEventListPageDataServices{
		TransactionService: transactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetEventItemPageDataRepositories{
		Event: repositories.Event,
	}
	itemPageDataServices := GetEventItemPageDataServices{
		TransactionService: transactionService,
		TranslationService: services.TranslationService,
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
func NewUseCasesUngrouped(eventRepo eventpb.EventDomainServiceServer, transactionService ports.TransactionService) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventRepositories{
		Event: eventRepo,
	}

	services := EventServices{
		AuthorizationService: nil, // Will be injected later by container
		TransactionService:   transactionService,
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services, transactionService)
}
