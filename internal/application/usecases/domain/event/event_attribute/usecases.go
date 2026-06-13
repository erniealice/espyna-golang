package event_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	readServices := ReadEventAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
		Event:          repositories.Event,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateEventAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventAttributeRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	deleteServices := DeleteEventAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventAttributesRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	listServices := ListEventAttributesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listPageDataRepos := GetEventAttributeListPageDataRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	listPageDataServices := GetEventAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	itemPageDataRepos := GetEventAttributeItemPageDataRepositories{
		EventAttribute: repositories.EventAttribute,
	}
	itemPageDataServices := GetEventAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
