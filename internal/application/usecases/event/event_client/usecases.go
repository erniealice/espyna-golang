package eventclient

import (
	"leapfor.xyz/espyna/internal/application/ports"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	readServices := ReadEventClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	updateServices := UpdateEventClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventClientRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	deleteServices := DeleteEventClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventClientsRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	listServices := ListEventClientsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEventClientListPageDataRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	getListPageDataServices := GetEventClientListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetEventClientItemPageDataRepositories{
		EventClient: repositories.EventClient,
		Event:       repositories.Event,
		Client:      repositories.Client,
	}
	getItemPageDataServices := GetEventClientItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}

	services := EventClientServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
