package eventattendee

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// DeleteEventAttendeeRepositories groups all repository dependencies
type DeleteEventAttendeeRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// DeleteEventAttendeeServices groups all business service dependencies
type DeleteEventAttendeeServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteEventAttendeeUseCase handles the business logic for deleting event attendee associations
type DeleteEventAttendeeUseCase struct {
	repositories DeleteEventAttendeeRepositories
	services     DeleteEventAttendeeServices
}

// NewDeleteEventAttendeeUseCase creates a new DeleteEventAttendeeUseCase
func NewDeleteEventAttendeeUseCase(
	repositories DeleteEventAttendeeRepositories,
	services DeleteEventAttendeeServices,
) *DeleteEventAttendeeUseCase {
	return &DeleteEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteEventAttendeeUseCaseUngrouped creates a new DeleteEventAttendeeUseCase
// Deprecated: Use NewDeleteEventAttendeeUseCase with grouped parameters instead
func NewDeleteEventAttendeeUseCaseUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
) *DeleteEventAttendeeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteEventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         nil,
	}

	services := DeleteEventAttendeeServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &DeleteEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event attendee operation
func (uc *DeleteEventAttendeeUseCase) Execute(ctx context.Context, req *eventattendeepb.DeleteEventAttendeeRequest) (*eventattendeepb.DeleteEventAttendeeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventAttendee,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventAttendee, entityid.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventAttendee.DeleteEventAttendee(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteEventAttendeeUseCase) validateInput(req *eventattendeepb.DeleteEventAttendeeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event attendee data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event attendee ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteEventAttendeeUseCase) validateBusinessRules(eventAttendee *eventattendeepb.EventAttendee) error {
	// Additional business rules can be added here
	// - Check if event attendee association can be safely deleted
	// - Validate impact on event capacity
	// - Check for related records that might be affected

	return nil
}
