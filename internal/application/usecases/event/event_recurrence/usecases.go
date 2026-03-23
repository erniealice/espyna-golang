package eventrecurrence

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// EventRecurrenceRepositories groups all repository dependencies for event recurrence use cases
type EventRecurrenceRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// EventRecurrenceServices groups all business service dependencies for event recurrence use cases
type EventRecurrenceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all event recurrence-related use cases
type UseCases struct {
	CreateEventRecurrence          *CreateEventRecurrenceUseCase
	ReadEventRecurrence            *ReadEventRecurrenceUseCase
	UpdateEventRecurrence          *UpdateEventRecurrenceUseCase
	DeleteEventRecurrence          *DeleteEventRecurrenceUseCase
	ListEventRecurrences           *ListEventRecurrencesUseCase
	GetEventRecurrenceListPageData *GetEventRecurrenceListPageDataUseCase
	GetEventRecurrenceItemPageData *GetEventRecurrenceItemPageDataUseCase
}

// NewUseCases creates a new collection of event recurrence use cases
func NewUseCases(
	repositories EventRecurrenceRepositories,
	services EventRecurrenceServices,
	transactionService ports.TransactionService,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventRecurrenceRepositories(repositories)
	createServices := CreateEventRecurrenceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventRecurrenceRepositories(repositories)
	readServices := ReadEventRecurrenceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventRecurrenceRepositories(repositories)
	updateServices := UpdateEventRecurrenceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventRecurrenceRepositories(repositories)
	deleteServices := DeleteEventRecurrenceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventRecurrencesRepositories(repositories)
	listServices := ListEventRecurrencesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   transactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetEventRecurrenceListPageDataRepositories{
		EventRecurrence: repositories.EventRecurrence,
	}
	listPageDataServices := GetEventRecurrenceListPageDataServices{
		TransactionService: transactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetEventRecurrenceItemPageDataRepositories{
		EventRecurrence: repositories.EventRecurrence,
	}
	itemPageDataServices := GetEventRecurrenceItemPageDataServices{
		TransactionService: transactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateEventRecurrence:          NewCreateEventRecurrenceUseCase(createRepos, createServices),
		ReadEventRecurrence:            NewReadEventRecurrenceUseCase(readRepos, readServices),
		UpdateEventRecurrence:          NewUpdateEventRecurrenceUseCase(updateRepos, updateServices),
		DeleteEventRecurrence:          NewDeleteEventRecurrenceUseCase(deleteRepos, deleteServices),
		ListEventRecurrences:           NewListEventRecurrencesUseCase(listRepos, listServices),
		GetEventRecurrenceListPageData: NewGetEventRecurrenceListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetEventRecurrenceItemPageData: NewGetEventRecurrenceItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event recurrence use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer, transactionService ports.TransactionService) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := EventRecurrenceServices{
		AuthorizationService: nil, // Will be injected later by container
		TransactionService:   transactionService,
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services, transactionService)
}
