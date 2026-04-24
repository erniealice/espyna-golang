package eventtagassignment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// ReadEventTagAssignmentRepositories groups all repository dependencies
type ReadEventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
}

// ReadEventTagAssignmentServices groups all business service dependencies
type ReadEventTagAssignmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadEventTagAssignmentUseCase handles the business logic for reading event_tag_assignments
type ReadEventTagAssignmentUseCase struct {
	repositories ReadEventTagAssignmentRepositories
	services     ReadEventTagAssignmentServices
}

// NewReadEventTagAssignmentUseCase creates use case with grouped dependencies
func NewReadEventTagAssignmentUseCase(
	repositories ReadEventTagAssignmentRepositories,
	services ReadEventTagAssignmentServices,
) *ReadEventTagAssignmentUseCase {
	return &ReadEventTagAssignmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event_tag_assignment operation
func (uc *ReadEventTagAssignmentUseCase) Execute(ctx context.Context, req *eventtagassignmentpb.ReadEventTagAssignmentRequest) (*eventtagassignmentpb.ReadEventTagAssignmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTagAssignment, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventTagAssignment, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag_assignment.errors.authorization_failed", "Authorization failed for event_tag_assignment")
		return nil, errors.New(translatedError)
	}

	return uc.repositories.EventTagAssignment.ReadEventTagAssignment(ctx, req)
}
