package eventrecurrence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// CreateEventRecurrenceRepositories groups all repository dependencies
type CreateEventRecurrenceRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// CreateEventRecurrenceServices groups all business service dependencies
type CreateEventRecurrenceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEventRecurrenceUseCase handles the business logic for creating event recurrences
type CreateEventRecurrenceUseCase struct {
	repositories CreateEventRecurrenceRepositories
	services     CreateEventRecurrenceServices
}

// NewCreateEventRecurrenceUseCase creates use case with grouped dependencies
func NewCreateEventRecurrenceUseCase(
	repositories CreateEventRecurrenceRepositories,
	services CreateEventRecurrenceServices,
) *CreateEventRecurrenceUseCase {
	return &CreateEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventRecurrenceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventRecurrenceUseCase with grouped parameters instead
func NewCreateEventRecurrenceUseCaseUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer) *CreateEventRecurrenceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateEventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := CreateEventRecurrenceServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return &CreateEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event recurrence operation
func (uc *CreateEventRecurrenceUseCase) Execute(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"event_recurrence", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event recurrence creation within a transaction
func (uc *CreateEventRecurrenceUseCase) executeWithTransaction(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	var result *eventrecurrencepb.CreateEventRecurrenceResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "event_recurrence.errors.creation_failed", "Event recurrence creation failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *CreateEventRecurrenceUseCase) executeCore(ctx context.Context, req *eventrecurrencepb.CreateEventRecurrenceRequest) (*eventrecurrencepb.CreateEventRecurrenceResponse, error) {
	// Business rule: Required fields validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.data_required", "Event recurrence data is required [DEFAULT]"))
	}

	// Business enrichment (must happen before validation to auto-generate timestamps)
	enrichedEventRecurrence := uc.applyBusinessLogic(req.Data)

	// Business validation
	if err := uc.validateBusinessRules(ctx, enrichedEventRecurrence); err != nil {
		return nil, err
	}

	// Delegate to repository
	return uc.repositories.EventRecurrence.CreateEventRecurrence(ctx, &eventrecurrencepb.CreateEventRecurrenceRequest{
		Data: enrichedEventRecurrence,
	})
}

// applyBusinessLogic applies business rules and returns enriched event recurrence
func (uc *CreateEventRecurrenceUseCase) applyBusinessLogic(eventRecurrence *eventrecurrencepb.EventRecurrence) *eventrecurrencepb.EventRecurrence {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if eventRecurrence.Id == "" {
		if uc.services.IDService != nil {
			eventRecurrence.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			eventRecurrence.Id = fmt.Sprintf("event_recurrence-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new event recurrences
	eventRecurrence.Active = true

	// Business logic: Set creation audit fields
	ts := now.UnixMilli()
	tsStr := now.Format(time.RFC3339)
	eventRecurrence.DateCreated = &ts
	eventRecurrence.DateCreatedString = &tsStr
	eventRecurrence.DateModified = &ts
	eventRecurrence.DateModifiedString = &tsStr

	return eventRecurrence
}

// validateBusinessRules enforces business constraints with translated error messages
func (uc *CreateEventRecurrenceUseCase) validateBusinessRules(ctx context.Context, eventRecurrence *eventrecurrencepb.EventRecurrence) error {
	// Business rule: name is required
	if eventRecurrence.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.name_required", "Event recurrence name is required [DEFAULT]"))
	}

	// Business rule: rrule_string is required (source of truth for the recurrence pattern)
	if eventRecurrence.RruleString == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.rrule_string_required", "RRULE string is required [DEFAULT]"))
	}

	// Business rule: workspace_id is required (tenant scope)
	if eventRecurrence.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.workspace_id_required", "Workspace ID is required [DEFAULT]"))
	}

	return nil
}
