package eventtagassignment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// UseCases contains all event_tag_assignment-related use cases
type UseCases struct {
	CreateEventTagAssignment       *CreateEventTagAssignmentUseCase
	ReadEventTagAssignment         *ReadEventTagAssignmentUseCase
	DeleteEventTagAssignment       *DeleteEventTagAssignmentUseCase
	ListEventTagAssignments        *ListEventTagAssignmentsUseCase
	ListEventTagAssignmentsByEvent *ListEventTagAssignmentsByEventUseCase
	SetEventTagAssignments         *SetEventTagAssignmentsUseCase
}

// EventTagAssignmentRepositories groups all repository dependencies for event_tag_assignment use cases
type EventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
	Event              eventpb.EventDomainServiceServer
	EventTag           eventtagpb.EventTagDomainServiceServer
}

// EventTagAssignmentServices groups all business service dependencies for event_tag_assignment use cases
type EventTagAssignmentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of event_tag_assignment use cases
func NewUseCases(
	repositories EventTagAssignmentRepositories,
	services EventTagAssignmentServices,
) *UseCases {
	createRepos := CreateEventTagAssignmentRepositories{
		EventTagAssignment: repositories.EventTagAssignment,
		Event:              repositories.Event,
		EventTag:           repositories.EventTag,
	}
	createServices := CreateEventTagAssignmentServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEventTagAssignmentRepositories{EventTagAssignment: repositories.EventTagAssignment}
	readServices := ReadEventTagAssignmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteEventTagAssignmentRepositories{EventTagAssignment: repositories.EventTagAssignment}
	deleteServices := DeleteEventTagAssignmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListEventTagAssignmentsRepositories{EventTagAssignment: repositories.EventTagAssignment}
	listServices := ListEventTagAssignmentsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	setRepos := SetEventTagAssignmentsRepositories{
		EventTagAssignment: repositories.EventTagAssignment,
		Event:              repositories.Event,
		EventTag:           repositories.EventTag,
	}
	setServices := SetEventTagAssignmentsServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	listUC := NewListEventTagAssignmentsUseCase(listRepos, listServices)

	return &UseCases{
		CreateEventTagAssignment:       NewCreateEventTagAssignmentUseCase(createRepos, createServices),
		ReadEventTagAssignment:         NewReadEventTagAssignmentUseCase(readRepos, readServices),
		DeleteEventTagAssignment:       NewDeleteEventTagAssignmentUseCase(deleteRepos, deleteServices),
		ListEventTagAssignments:        listUC,
		ListEventTagAssignmentsByEvent: NewListEventTagAssignmentsByEventUseCase(listUC),
		SetEventTagAssignments:         NewSetEventTagAssignmentsUseCase(setRepos, setServices),
	}
}
