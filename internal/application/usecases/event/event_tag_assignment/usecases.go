package eventtagassignment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// UseCases contains all event_tag_assignment-related use cases
type UseCases struct {
	CreateEventTagAssignment        *CreateEventTagAssignmentUseCase
	ReadEventTagAssignment          *ReadEventTagAssignmentUseCase
	DeleteEventTagAssignment        *DeleteEventTagAssignmentUseCase
	ListEventTagAssignments         *ListEventTagAssignmentsUseCase
	ListEventTagAssignmentsByEvent  *ListEventTagAssignmentsByEventUseCase
	SetEventTagAssignments          *SetEventTagAssignmentsUseCase
}

// EventTagAssignmentRepositories groups all repository dependencies for event_tag_assignment use cases
type EventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
	Event              eventpb.EventDomainServiceServer
	EventTag           eventtagpb.EventTagDomainServiceServer
}

// EventTagAssignmentServices groups all business service dependencies for event_tag_assignment use cases
type EventTagAssignmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEventTagAssignmentRepositories{EventTagAssignment: repositories.EventTagAssignment}
	readServices := ReadEventTagAssignmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteEventTagAssignmentRepositories{EventTagAssignment: repositories.EventTagAssignment}
	deleteServices := DeleteEventTagAssignmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEventTagAssignmentsRepositories{EventTagAssignment: repositories.EventTagAssignment}
	listServices := ListEventTagAssignmentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	setRepos := SetEventTagAssignmentsRepositories{
		EventTagAssignment: repositories.EventTagAssignment,
		Event:              repositories.Event,
		EventTag:           repositories.EventTag,
	}
	setServices := SetEventTagAssignmentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
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
