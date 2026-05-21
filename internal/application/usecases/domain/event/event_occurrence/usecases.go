package eventoccurrence

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
)

// EventOccurrenceRepositories groups all repository dependencies for event occurrence use cases
type EventOccurrenceRepositories struct {
	EventOccurrence eventoccurrencepb.EventOccurrenceDomainServiceServer // Primary entity repository
}

// EventOccurrenceServices groups all business service dependencies for event occurrence use cases
type EventOccurrenceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UseCases contains all event occurrence-related use cases.
// Read-only: populated by the background recurrence expansion job, not by user CRUD.
type UseCases struct {
	ListEventOccurrences           *ListEventOccurrencesUseCase
	GetEventOccurrenceListPageData *GetEventOccurrenceListPageDataUseCase
	GetEventOccurrenceItemPageData *GetEventOccurrenceItemPageDataUseCase
}

// NewUseCases creates a new collection of event occurrence use cases
func NewUseCases(
	repositories EventOccurrenceRepositories,
	services EventOccurrenceServices,
) *UseCases {
	listRepos := ListEventOccurrencesRepositories{
		EventOccurrence: repositories.EventOccurrence,
	}
	listServices := ListEventOccurrencesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetEventOccurrenceListPageDataRepositories{
		EventOccurrence: repositories.EventOccurrence,
	}
	listPageDataServices := GetEventOccurrenceListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetEventOccurrenceItemPageDataRepositories{
		EventOccurrence: repositories.EventOccurrence,
	}
	itemPageDataServices := GetEventOccurrenceItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		ListEventOccurrences:           NewListEventOccurrencesUseCase(listRepos, listServices),
		GetEventOccurrenceListPageData: NewGetEventOccurrenceListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetEventOccurrenceItemPageData: NewGetEventOccurrenceItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
