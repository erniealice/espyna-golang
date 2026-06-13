package eventtagassignment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// DeleteEventTagAssignmentRepositories groups all repository dependencies
type DeleteEventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
}

// DeleteEventTagAssignmentServices groups all business service dependencies
type DeleteEventTagAssignmentServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteEventTagAssignmentUseCase handles the business logic for deleting event_tag_assignments
type DeleteEventTagAssignmentUseCase struct {
	repositories DeleteEventTagAssignmentRepositories
	services     DeleteEventTagAssignmentServices
}

// NewDeleteEventTagAssignmentUseCase creates use case with grouped dependencies
func NewDeleteEventTagAssignmentUseCase(
	repositories DeleteEventTagAssignmentRepositories,
	services DeleteEventTagAssignmentServices,
) *DeleteEventTagAssignmentUseCase {
	return &DeleteEventTagAssignmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event_tag_assignment operation
func (uc *DeleteEventTagAssignmentUseCase) Execute(ctx context.Context, req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) (*eventtagassignmentpb.DeleteEventTagAssignmentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventTagAssignment,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventTagAssignment, entityid.ActionDelete)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	return uc.repositories.EventTagAssignment.DeleteEventTagAssignment(ctx, req)
}

func (uc *DeleteEventTagAssignmentUseCase) validateInput(req *eventtagassignmentpb.DeleteEventTagAssignmentRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event_tag_assignment data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event_tag_assignment ID is required")
	}
	return nil
}
