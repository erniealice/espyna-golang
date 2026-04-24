package eventtag

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// UseCases contains all event_tag-related use cases
type UseCases struct {
	CreateEventTag          *CreateEventTagUseCase
	ReadEventTag            *ReadEventTagUseCase
	UpdateEventTag          *UpdateEventTagUseCase
	DeleteEventTag          *DeleteEventTagUseCase
	ListEventTags           *ListEventTagsUseCase
	GetEventTagListPageData *GetEventTagListPageDataUseCase
	GetEventTagItemPageData *GetEventTagItemPageDataUseCase
}

// EventTagRepositories groups all repository dependencies for event_tag use cases
type EventTagRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// EventTagServices groups all business service dependencies for event_tag use cases
type EventTagServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of event_tag use cases
func NewUseCases(
	repositories EventTagRepositories,
	services EventTagServices,
) *UseCases {
	createRepos := CreateEventTagRepositories{EventTag: repositories.EventTag}
	createServices := CreateEventTagServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventTagRepositories{EventTag: repositories.EventTag}
	readServices := ReadEventTagServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventTagRepositories{EventTag: repositories.EventTag}
	updateServices := UpdateEventTagServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventTagRepositories{EventTag: repositories.EventTag}
	deleteServices := DeleteEventTagServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventTagsRepositories{EventTag: repositories.EventTag}
	listServices := ListEventTagsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEventTagListPageDataRepositories{EventTag: repositories.EventTag}
	getListPageDataServices := GetEventTagListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetEventTagItemPageDataRepositories{EventTag: repositories.EventTag}
	getItemPageDataServices := GetEventTagItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateEventTag:          NewCreateEventTagUseCase(createRepos, createServices),
		ReadEventTag:            NewReadEventTagUseCase(readRepos, readServices),
		UpdateEventTag:          NewUpdateEventTagUseCase(updateRepos, updateServices),
		DeleteEventTag:          NewDeleteEventTagUseCase(deleteRepos, deleteServices),
		ListEventTags:           NewListEventTagsUseCase(listRepos, listServices),
		GetEventTagListPageData: NewGetEventTagListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetEventTagItemPageData: NewGetEventTagItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
