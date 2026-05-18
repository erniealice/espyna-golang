package eventtagassignment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// DeleteEventTagAssignmentRepositories groups all repository dependencies
type DeleteEventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
}

// DeleteEventTagAssignmentServices groups all business service dependencies
type DeleteEventTagAssignmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTagAssignment, ports.ActionDelete); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventTagAssignment, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
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
