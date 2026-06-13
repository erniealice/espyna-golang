package eventclient

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// UseCases contains all event client-related use cases
type UseCases struct {
	CreateEventClient          *CreateEventClientUseCase
	ReadEventClient            *ReadEventClientUseCase
	UpdateEventClient          *UpdateEventClientUseCase
	DeleteEventClient          *DeleteEventClientUseCase
	ListEventClients           *ListEventClientsUseCase
	GetEventClientListPageData *GetEventClientListPageDataUseCase
	GetEventClientItemPageData *GetEventClientItemPageDataUseCase
}

// EventClientRepositories groups all repository dependencies for event client use cases
type EventClientRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// EventClientServices groups all business service dependencies for event client use cases
type EventClientServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event client use cases
func NewUseCases(
	repositories EventClientRepositories,
	services EventClientServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	createServices := CreateEventClientServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	readServices := ReadEventClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	updateServices := UpdateEventClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	deleteServices := DeleteEventClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventClientsRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	listServices := ListEventClientsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetEventClientListPageDataRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	getListPageDataServices := GetEventClientListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetEventClientItemPageDataRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	getItemPageDataServices := GetEventClientItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateEventClient:          NewCreateEventClientUseCase(createRepos, createServices),
		ReadEventClient:            NewReadEventClientUseCase(readRepos, readServices),
		UpdateEventClient:          NewUpdateEventClientUseCase(updateRepos, updateServices),
		DeleteEventClient:          NewDeleteEventClientUseCase(deleteRepos, deleteServices),
		ListEventClients:           NewListEventClientsUseCase(listRepos, listServices),
		GetEventClientListPageData: NewGetEventClientListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetEventClientItemPageData: NewGetEventClientItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of event client use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	eventClientRepo eventclientpb.EventClientDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}

	services := EventClientServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
