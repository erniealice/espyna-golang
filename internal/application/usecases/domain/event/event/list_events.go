package event

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// ListEventsRepositories groups all repository dependencies
type ListEventsRepositories struct {
	Event eventpb.EventDomainServiceServer // Primary entity repository
}

// ListEventsServices groups all business service dependencies
type ListEventsServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListEventsUseCase handles the business logic for listing events
type ListEventsUseCase struct {
	repositories ListEventsRepositories
	services     ListEventsServices
}

// NewListEventsUseCase creates use case with grouped dependencies
func NewListEventsUseCase(
	repositories ListEventsRepositories,
	services ListEventsServices,
) *ListEventsUseCase {
	return &ListEventsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListEventsUseCase with grouped parameters instead
func NewListEventsUseCaseUngrouped(eventRepo eventpb.EventDomainServiceServer) *ListEventsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListEventsRepositories{
		Event: eventRepo,
	}

	services := ListEventsServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list events operation
func (uc *ListEventsUseCase) Execute(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Event, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &eventpb.ListEventsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Event.ListEvents(ctx, req)
	if err != nil {
		errorMessage := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event.errors.list_failed", "Failed to retrieve events [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	// Business logic post-processing (if needed)
	// Currently no additional business rules for list operation

	return resp, nil
}
