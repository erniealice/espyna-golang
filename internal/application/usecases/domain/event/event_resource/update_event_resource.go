package eventresource

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// UpdateEventResourceRepositories groups all repository dependencies
type UpdateEventResourceRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// UpdateEventResourceServices groups all business service dependencies
type UpdateEventResourceServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateEventResourceUseCase handles the business logic for updating event resource assignments
type UpdateEventResourceUseCase struct {
	repositories UpdateEventResourceRepositories
	services     UpdateEventResourceServices
}

// NewUpdateEventResourceUseCase creates a new UpdateEventResourceUseCase
func NewUpdateEventResourceUseCase(
	repositories UpdateEventResourceRepositories,
	services UpdateEventResourceServices,
) *UpdateEventResourceUseCase {
	return &UpdateEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateEventResourceUseCaseUngrouped creates a new UpdateEventResourceUseCase
// Deprecated: Use NewUpdateEventResourceUseCase with grouped parameters instead
func NewUpdateEventResourceUseCaseUngrouped(
	eventResourceRepo eventresourcepb.EventResourceDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
) *UpdateEventResourceUseCase {
	repositories := UpdateEventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         eventRepo,
	}

	services := UpdateEventResourceServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return &UpdateEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event resource operation
func (uc *UpdateEventResourceUseCase) Execute(ctx context.Context, req *eventresourcepb.UpdateEventResourceRequest) (*eventresourcepb.UpdateEventResourceResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventResource,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventResource, entityid.ActionUpdate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichEventResourceData(req.Data); err != nil {
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
	return uc.repositories.EventResource.UpdateEventResource(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateEventResourceUseCase) validateInput(req *eventresourcepb.UpdateEventResourceRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event resource data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event resource ID is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ResourceId == "" {
		return errors.New("resource ID is required")
	}
	return nil
}

// enrichEventResourceData adds audit information for updates
func (uc *UpdateEventResourceUseCase) enrichEventResourceData(eventResource *eventresourcepb.EventResource) error {
	now := time.Now()

	// Update audit fields
	eventResource.DateModified = &[]int64{now.UnixMilli()}[0]
	eventResource.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateEventResourceUseCase) validateBusinessRules(eventResource *eventresourcepb.EventResource) error {
	// Validate that event and resource IDs are not the same
	if eventResource.EventId == eventResource.ResourceId {
		return errors.New("event ID and resource ID cannot be the same")
	}

	// Additional business rules can be added here
	// - Check resource type is still valid for the event
	// - Validate updated assignment status transitions

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateEventResourceUseCase) validateEntityReferences(ctx context.Context, eventResource *eventresourcepb.EventResource) error {
	// Validate Event entity reference
	if eventResource.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventResource.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventResource.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventResource.EventId)
		}
	}

	return nil
}
