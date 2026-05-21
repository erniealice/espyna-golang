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
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
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
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetEventOccurrenceListPageDataRepositories{
		EventOccurrence: repositories.EventOccurrence,
	}
	listPageDataServices := GetEventOccurrenceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetEventOccurrenceItemPageDataRepositories{
		EventOccurrence: repositories.EventOccurrence,
	}
	itemPageDataServices := GetEventOccurrenceItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		ListEventOccurrences:           NewListEventOccurrencesUseCase(listRepos, listServices),
		GetEventOccurrenceListPageData: NewGetEventOccurrenceListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetEventOccurrenceItemPageData: NewGetEventOccurrenceItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
