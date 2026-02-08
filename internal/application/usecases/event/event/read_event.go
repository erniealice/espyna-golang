package event

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// ReadEventRepositories groups all repository dependencies
type ReadEventRepositories struct {
	Event eventpb.EventDomainServiceServer // Primary entity repository
}

// ReadEventServices groups all business service dependencies
type ReadEventServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadEventUseCase handles the business logic for reading a single event
type ReadEventUseCase struct {
	repositories ReadEventRepositories
	services     ReadEventServices
}

// NewReadEventUseCase creates use case with grouped dependencies
func NewReadEventUseCase(
	repositories ReadEventRepositories,
	services ReadEventServices,
) *ReadEventUseCase {
	return &ReadEventUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventUseCase with grouped parameters instead
func NewReadEventUseCaseUngrouped(eventRepo eventpb.EventDomainServiceServer) *ReadEventUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadEventRepositories{
		Event: eventRepo,
	}

	services := ReadEventServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ReadEventUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event operation
func (uc *ReadEventUseCase) Execute(ctx context.Context, req *eventpb.ReadEventRequest) (*eventpb.ReadEventResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if uc.services.AuthorizationService != nil {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		authorized, err := uc.services.AuthorizationService.HasPermission(ctx, userID, "event_read")
		if err != nil || !authorized {
			authError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.errors.authorization_failed", "Authorization failed for academic events [DEFAULT]")
			return nil, errors.New(authError)
		}
	}

	// Call repository
	resp, err := uc.repositories.Event.ReadEvent(ctx, req)
	if err != nil {
		// Check if this is a not found error from repository
		if contains := contextutil.Contains(err.Error(), "not found"); contains {
			errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event.errors.not_found", map[string]interface{}{"eventId": req.Data.Id}, "Event not found [DEFAULT]")
			return nil, errors.New(errorMessage)
		}
		return nil, err
	}

	// Business logic validation
	if len(resp.Data) == 0 {
		errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event.errors.not_found", map[string]interface{}{"eventId": req.Data.Id}, "Event not found [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadEventUseCase) validateInput(ctx context.Context, req *eventpb.ReadEventRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.data_required", "Academic event data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.id_required", "Event ID is required [DEFAULT]"))
	}
	return nil
}
