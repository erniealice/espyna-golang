package eventresource

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// UseCases contains all event resource-related use cases
type UseCases struct {
	CreateEventResource          *CreateEventResourceUseCase
	ReadEventResource            *ReadEventResourceUseCase
	UpdateEventResource          *UpdateEventResourceUseCase
	DeleteEventResource          *DeleteEventResourceUseCase
	ListEventResources           *ListEventResourcesUseCase
	GetEventResourceListPageData *GetEventResourceListPageDataUseCase
	GetEventResourceItemPageData *GetEventResourceItemPageDataUseCase
}

// EventResourceRepositories groups all repository dependencies for event resource use cases
type EventResourceRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// EventResourceServices groups all business service dependencies for event resource use cases
type EventResourceServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event resource use cases
func NewUseCases(
	repositories EventResourceRepositories,
	services EventResourceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	createServices := CreateEventResourceServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	readServices := ReadEventResourceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	updateServices := UpdateEventResourceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	deleteServices := DeleteEventResourceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventResourcesRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	listServices := ListEventResourcesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetEventResourceListPageDataRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	getListPageDataServices := GetEventResourceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetEventResourceItemPageDataRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	getItemPageDataServices := GetEventResourceItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateEventResource:          NewCreateEventResourceUseCase(createRepos, createServices),
		ReadEventResource:            NewReadEventResourceUseCase(readRepos, readServices),
		UpdateEventResource:          NewUpdateEventResourceUseCase(updateRepos, updateServices),
		DeleteEventResource:          NewDeleteEventResourceUseCase(deleteRepos, deleteServices),
		ListEventResources:           NewListEventResourcesUseCase(listRepos, listServices),
		GetEventResourceListPageData: NewGetEventResourceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetEventResourceItemPageData: NewGetEventResourceItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event resource use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	eventResourceRepo eventresourcepb.EventResourceDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         eventRepo,
	}

	services := EventResourceServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
