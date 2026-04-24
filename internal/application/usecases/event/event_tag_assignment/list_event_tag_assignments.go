package eventtagassignment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// ListEventTagAssignmentsRepositories groups all repository dependencies
type ListEventTagAssignmentsRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
}

// ListEventTagAssignmentsServices groups all business service dependencies
type ListEventTagAssignmentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListEventTagAssignmentsUseCase handles the business logic for listing event_tag_assignments
type ListEventTagAssignmentsUseCase struct {
	repositories ListEventTagAssignmentsRepositories
	services     ListEventTagAssignmentsServices
}

// NewListEventTagAssignmentsUseCase creates use case with grouped dependencies
func NewListEventTagAssignmentsUseCase(
	repositories ListEventTagAssignmentsRepositories,
	services ListEventTagAssignmentsServices,
) *ListEventTagAssignmentsUseCase {
	return &ListEventTagAssignmentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event_tag_assignments operation.
// Filters are passed through; the infra layer is responsible for applying them.
func (uc *ListEventTagAssignmentsUseCase) Execute(ctx context.Context, req *eventtagassignmentpb.ListEventTagAssignmentsRequest) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTagAssignment, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventTagAssignment, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	if req == nil {
		req = &eventtagassignmentpb.ListEventTagAssignmentsRequest{}
	}

	return uc.repositories.EventTagAssignment.ListEventTagAssignments(ctx, req)
}
