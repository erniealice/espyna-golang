package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// CreateEventRepositories groups all repository dependencies
type CreateEventRepositories struct {
	Event eventpb.EventDomainServiceServer // Primary entity repository
}

// CreateEventServices groups all business service dependencies
type CreateEventServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEventUseCase handles the business logic for creating events
type CreateEventUseCase struct {
	repositories CreateEventRepositories
	services     CreateEventServices
}

// NewCreateEventUseCase creates use case with grouped dependencies
func NewCreateEventUseCase(
	repositories CreateEventRepositories,
	services CreateEventServices,
) *CreateEventUseCase {
	return &CreateEventUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventUseCase with grouped parameters instead
func NewCreateEventUseCaseUngrouped(eventRepo eventpb.EventDomainServiceServer) *CreateEventUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateEventRepositories{
		Event: eventRepo,
	}

	services := CreateEventServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return &CreateEventUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event operation
func (uc *CreateEventUseCase) Execute(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event creation within a transaction
func (uc *CreateEventUseCase) executeWithTransaction(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	var result *eventpb.CreateEventResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "event.errors.creation_failed", "Event creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateEventUseCase) executeCore(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	// Business rule: Required fields validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.request_required", "Request is required for academic events [DEFAULT]"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.data_required", "Academic event data is required [DEFAULT]"))
	}

	// Authorization check
	if uc.services.AuthorizationService != nil {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		authorized, err := uc.services.AuthorizationService.HasPermission(ctx, userID, "event_create")
		if err != nil || !authorized {
			authError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.errors.authorization_failed", "Authorization failed for academic events [DEFAULT]")
			return nil, errors.New(authError)
		}
	}

	// Business enrichment (must happen before validation to auto-generate timestamps)
	enrichedEvent := uc.applyBusinessLogic(req.Data)

	// Business validation
	if err := uc.validateBusinessRules(ctx, enrichedEvent); err != nil {
		return nil, err
	}

	// Delegate to repository
	return uc.repositories.Event.CreateEvent(ctx, &eventpb.CreateEventRequest{
		Data: enrichedEvent,
	})
}

// applyBusinessLogic applies business rules and returns enriched event
func (uc *CreateEventUseCase) applyBusinessLogic(event *eventpb.Event) *eventpb.Event {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if event.Id == "" {
		if uc.services.IDService != nil {
			event.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			event.Id = fmt.Sprintf("event-%d", now.UnixNano())
		}
	}

	// Business logic: Auto-generate valid time fields if missing
	if event.StartDateTimeUtc == 0 {
		// Set start time to 1 hour from now
		event.StartDateTimeUtc = now.Add(1 * time.Hour).UnixMilli()
	}
	if event.EndDateTimeUtc == 0 {
		// Set end time to 1 hour after start time
		event.EndDateTimeUtc = event.StartDateTimeUtc + 3600*1000
	}

	// Business logic: Set default timezone if not provided
	if event.Timezone == "" {
		event.Timezone = "UTC"
	}

	// Business logic: Set active status for new events
	event.Active = true

	// Business logic: Set creation audit fields
	event.DateCreated = &[]int64{now.Unix()}[0]
	event.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	event.DateModified = &[]int64{now.Unix()}[0]
	event.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return event
}

// validateBusinessRules enforces business constraints with translated error messages
func (uc *CreateEventUseCase) validateBusinessRules(ctx context.Context, event *eventpb.Event) error {
	// Business rule: Required fields validation
	if event.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.name_required", "Event name is required [DEFAULT]"))
	}

	// Business rule: Event timing logic
	if event.StartDateTimeUtc >= event.EndDateTimeUtc {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.invalid_time_range", "Event start time must be before end time [DEFAULT]"))
	}

	// Business rule: No past events allowed (use current time for consistency)
	now := time.Now()
	if event.StartDateTimeUtc < now.UnixMilli() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.no_past_events", "Cannot create events in the past [DEFAULT]"))
	}

	// Business rule: Event duration constraints
	duration := event.EndDateTimeUtc - event.StartDateTimeUtc
	if duration < 300*1000 { // 5 minutes minimum
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.minimum_duration", "Event must be at least 5 minutes long [DEFAULT]"))
	}

	if duration > 86400*7*1000 { // 7 days maximum
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.maximum_duration", "Event cannot be longer than 7 days [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(event.Name) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.name_too_long", "Event name cannot exceed 255 characters [DEFAULT]"))
	}

	// Business rule: Description length constraints (if provided)
	if event.Description != nil && len(*event.Description) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event.validation.description_too_long", "Event description cannot exceed 2000 characters [DEFAULT]"))
	}

	return nil
}
