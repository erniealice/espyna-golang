package event_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

// EventAttributeRepositories groups all repository dependencies for event attribute use cases
type EventAttributeRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
	Event          eventpb.EventDomainServiceServer
	Attribute      attributepb.AttributeDomainServiceServer
}

// EventAttributeServices groups all business service dependencies for event attribute use cases
type EventAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all event attribute-related use cases
type UseCases struct {
	CreateEventAttribute          *CreateEventAttributeUseCase
	ReadEventAttribute            *ReadEventAttributeUseCase
	UpdateEventAttribute          *UpdateEventAttributeUseCase
	DeleteEventAttribute          *DeleteEventAttributeUseCase
	ListEventAttributes           *ListEventAttributesUseCase
	GetEventAttributeListPageData *GetEventAttributeListPageDataUseCase
	GetEventAttributeItemPageData *GetEventAttributeItemPageDataUseCase
}

// NewUseCases creates a new collection of event attribute use cases
func NewUseCases(
	repositories EventAttributeRepositories,
	services EventAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
		Event:          repositories.Event,
		Attribute:      repositories.Attribute,
	}
	createServices := CreateEventAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	readServices := ReadEventAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
		Event:          repositories.Event,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateEventAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	deleteServices := DeleteEventAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventAttributesRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	listServices := ListEventAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetEventAttributeListPageDataRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	listPageDataServices := GetEventAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetEventAttributeItemPageDataRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	itemPageDataServices := GetEventAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateEventAttribute:          NewCreateEventAttributeUseCase(createRepos, createServices),
		ReadEventAttribute:            NewReadEventAttributeUseCase(readRepos, readServices),
		UpdateEventAttribute:          NewUpdateEventAttributeUseCase(updateRepos, updateServices),
		DeleteEventAttribute:          NewDeleteEventAttributeUseCase(deleteRepos, deleteServices),
		ListEventAttributes:           NewListEventAttributesUseCase(listRepos, listServices),
		GetEventAttributeListPageData: NewGetEventAttributeListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetEventAttributeItemPageData: NewGetEventAttributeItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
