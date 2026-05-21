package eventattendee

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// ListEventAttendeesRepositories groups all repository dependencies
type ListEventAttendeesRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// ListEventAttendeesServices groups all business service dependencies
type ListEventAttendeesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ListEventAttendeesUseCase handles the business logic for listing event attendee associations
type ListEventAttendeesUseCase struct {
	repositories ListEventAttendeesRepositories
	services     ListEventAttendeesServices
}

// NewListEventAttendeesUseCase creates a new ListEventAttendeesUseCase
func NewListEventAttendeesUseCase(
	repositories ListEventAttendeesRepositories,
	services ListEventAttendeesServices,
) *ListEventAttendeesUseCase {
	return &ListEventAttendeesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventAttendeesUseCaseUngrouped creates a new ListEventAttendeesUseCase
// Deprecated: Use NewListEventAttendeesUseCase with grouped parameters instead
func NewListEventAttendeesUseCaseUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
) *ListEventAttendeesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListEventAttendeesRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         nil,
	}

	services := ListEventAttendeesServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventAttendeesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event attendees operation
func (uc *ListEventAttendeesUseCase) Execute(ctx context.Context, req *eventattendeepb.ListEventAttendeesRequest) (*eventattendeepb.ListEventAttendeesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityEventAttendee, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttendee, ports.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Handle nil request by creating default empty request for list operations
	if req == nil {
		req = &eventattendeepb.ListEventAttendeesRequest{}
	}

	// Call repository
	return uc.repositories.EventAttendee.ListEventAttendees(ctx, req)
}

// validateInput validates the input request
func (uc *ListEventAttendeesUseCase) validateInput(req *eventattendeepb.ListEventAttendeesRequest) error {
	// For list operations, nil request is allowed - we'll create a default empty request
	return nil
}
