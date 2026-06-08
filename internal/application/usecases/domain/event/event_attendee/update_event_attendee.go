package eventattendee

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// UpdateEventAttendeeRepositories groups all repository dependencies
type UpdateEventAttendeeRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// UpdateEventAttendeeServices groups all business service dependencies
type UpdateEventAttendeeServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// UpdateEventAttendeeUseCase handles the business logic for updating event attendee associations
type UpdateEventAttendeeUseCase struct {
	repositories UpdateEventAttendeeRepositories
	services     UpdateEventAttendeeServices
}

// NewUpdateEventAttendeeUseCase creates a new UpdateEventAttendeeUseCase
func NewUpdateEventAttendeeUseCase(
	repositories UpdateEventAttendeeRepositories,
	services UpdateEventAttendeeServices,
) *UpdateEventAttendeeUseCase {
	return &UpdateEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateEventAttendeeUseCaseUngrouped creates a new UpdateEventAttendeeUseCase
// Deprecated: Use NewUpdateEventAttendeeUseCase with grouped parameters instead
func NewUpdateEventAttendeeUseCaseUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
) *UpdateEventAttendeeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateEventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         eventRepo,
	}

	services := UpdateEventAttendeeServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &UpdateEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event attendee operation
func (uc *UpdateEventAttendeeUseCase) Execute(ctx context.Context, req *eventattendeepb.UpdateEventAttendeeRequest) (*eventattendeepb.UpdateEventAttendeeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventAttendee, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventAttendee, entityid.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichEventAttendeeData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventAttendee.UpdateEventAttendee(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateEventAttendeeUseCase) validateInput(req *eventattendeepb.UpdateEventAttendeeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event attendee data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event attendee ID is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	return nil
}

// enrichEventAttendeeData adds audit information for updates
func (uc *UpdateEventAttendeeUseCase) enrichEventAttendeeData(eventAttendee *eventattendeepb.EventAttendee) error {
	now := time.Now()

	// Update audit fields
	eventAttendee.DateModified = &[]int64{now.UnixMilli()}[0]
	eventAttendee.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateEventAttendeeUseCase) validateBusinessRules(eventAttendee *eventattendeepb.EventAttendee) error {
	// At least one attendee identity must be present
	hasClient := eventAttendee.ClientId != nil && *eventAttendee.ClientId != ""
	hasWorkspaceUser := eventAttendee.WorkspaceUserId != nil && *eventAttendee.WorkspaceUserId != ""

	if !hasClient && !hasWorkspaceUser && (eventAttendee.DisplayName == nil || *eventAttendee.DisplayName == "") {
		return errors.New("attendee must have a client_id, workspace_user_id, or display_name")
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateEventAttendeeUseCase) validateEntityReferences(ctx context.Context, eventAttendee *eventattendeepb.EventAttendee) error {
	// Validate Event entity reference
	if eventAttendee.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventAttendee.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventAttendee.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventAttendee.EventId)
		}
	}

	return nil
}
