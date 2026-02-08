package event

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// ListEventsRepositories groups all repository dependencies
type ListEventsRepositories struct {
	Event eventpb.EventDomainServiceServer // Primary entity repository
}

// ListEventsServices groups all business service dependencies
type ListEventsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ListEventsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list events operation
func (uc *ListEventsUseCase) Execute(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	// Input validation
	if req == nil {
		req = &eventpb.ListEventsRequest{}
	}

	// Authorization check
	if uc.services.AuthorizationService != nil {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		authorized, err := uc.services.AuthorizationService.HasPermission(ctx, userID, "event_list")
		if err != nil || !authorized {
			authError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.errors.authorization_failed", "Authorization failed for academic events [DEFAULT]")
			return nil, errors.New(authError)
		}
	}

	// Call repository
	resp, err := uc.repositories.Event.ListEvents(ctx, req)
	if err != nil {
		errorMessage := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.errors.list_failed", "Failed to retrieve events [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	// Business logic post-processing (if needed)
	// Currently no additional business rules for list operation

	return resp, nil
}
