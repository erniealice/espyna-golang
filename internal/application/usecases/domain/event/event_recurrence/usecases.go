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
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	transactionService ports.Transactor,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventRecurrenceRepositories(repositories)
	createServices := CreateEventRecurrenceServices{
		Authorizer:  services.Authorizer,
		Transactor:  transactionService,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventRecurrenceRepositories(repositories)
	readServices := ReadEventRecurrenceServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	updateRepos := UpdateEventRecurrenceRepositories(repositories)
	updateServices := UpdateEventRecurrenceServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	deleteRepos := DeleteEventRecurrenceRepositories(repositories)
	deleteServices := DeleteEventRecurrenceServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	listRepos := ListEventRecurrencesRepositories(repositories)
	listServices := ListEventRecurrencesServices{
		Authorizer: services.Authorizer,
		Transactor: transactionService,
		Translator: services.Translator,
	}

	listPageDataRepos := GetEventRecurrenceListPageDataRepositories{
		EventRecurrence: repositories.EventRecurrence,
	}
	listPageDataServices := GetEventRecurrenceListPageDataServices{
		Transactor: transactionService,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetEventRecurrenceItemPageDataRepositories{
		EventRecurrence: repositories.EventRecurrence,
	}
	itemPageDataServices := GetEventRecurrenceItemPageDataServices{
		Transactor: transactionService,
		Translator: services.Translator,
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
func NewUseCasesUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer, transactionService ports.Transactor) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := EventRecurrenceServices{
		Authorizer: nil, // Will be injected later by container
		Transactor: transactionService,
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services, transactionService)
}
