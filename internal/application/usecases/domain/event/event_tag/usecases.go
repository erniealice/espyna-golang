package eventtag

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event_tag use cases
func NewUseCases(
	repositories EventTagRepositories,
	services EventTagServices,
) *UseCases {
	createRepos := CreateEventTagRepositories{EventTag: repositories.EventTag}
	createServices := CreateEventTagServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventTagRepositories{EventTag: repositories.EventTag}
	readServices := ReadEventTagServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventTagRepositories{EventTag: repositories.EventTag}
	updateServices := UpdateEventTagServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventTagRepositories{EventTag: repositories.EventTag}
	deleteServices := DeleteEventTagServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventTagsRepositories{EventTag: repositories.EventTag}
	listServices := ListEventTagsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetEventTagListPageDataRepositories{EventTag: repositories.EventTag}
	getListPageDataServices := GetEventTagListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetEventTagItemPageDataRepositories{EventTag: repositories.EventTag}
	getItemPageDataServices := GetEventTagItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
