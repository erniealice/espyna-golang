package eventresource

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	readServices := ReadEventResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	updateServices := UpdateEventResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventResourceRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	deleteServices := DeleteEventResourceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventResourcesRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	listServices := ListEventResourcesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEventResourceListPageDataRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	getListPageDataServices := GetEventResourceListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetEventResourceItemPageDataRepositories{
		EventResource: repositories.EventResource,
		Event:         repositories.Event,
	}
	getItemPageDataServices := GetEventResourceItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := EventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         eventRepo,
	}

	services := EventResourceServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
