package eventrecurrence

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// DeleteEventRecurrenceRepositories groups all repository dependencies
type DeleteEventRecurrenceRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// DeleteEventRecurrenceServices groups all business service dependencies
type DeleteEventRecurrenceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteEventRecurrenceUseCase handles the business logic for deleting event recurrences
type DeleteEventRecurrenceUseCase struct {
	repositories DeleteEventRecurrenceRepositories
	services     DeleteEventRecurrenceServices
}

// NewDeleteEventRecurrenceUseCase creates use case with grouped dependencies
func NewDeleteEventRecurrenceUseCase(
	repositories DeleteEventRecurrenceRepositories,
	services DeleteEventRecurrenceServices,
) *DeleteEventRecurrenceUseCase {
	return &DeleteEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteEventRecurrenceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteEventRecurrenceUseCase with grouped parameters instead
func NewDeleteEventRecurrenceUseCaseUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer) *DeleteEventRecurrenceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteEventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := DeleteEventRecurrenceServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &DeleteEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event recurrence operation
func (uc *DeleteEventRecurrenceUseCase) Execute(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"event_recurrence", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event recurrence deletion within a transaction
func (uc *DeleteEventRecurrenceUseCase) executeWithTransaction(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	var result *eventrecurrencepb.DeleteEventRecurrenceResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "event_recurrence.errors.deletion_failed", "Event recurrence deletion failed [DEFAULT]")
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
func (uc *DeleteEventRecurrenceUseCase) executeCore(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) (*eventrecurrencepb.DeleteEventRecurrenceResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Check if event recurrence exists
	_, err := uc.getExistingEventRecurrence(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventRecurrence.DeleteEventRecurrence(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteEventRecurrenceUseCase) validateInput(ctx context.Context, req *eventrecurrencepb.DeleteEventRecurrenceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.data_required", "Event recurrence data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_recurrence.validation.id_required", "Event recurrence ID is required [DEFAULT]"))
	}
	return nil
}

// getExistingEventRecurrence retrieves the current event recurrence state
func (uc *DeleteEventRecurrenceUseCase) getExistingEventRecurrence(ctx context.Context, eventRecurrenceID string) (*eventrecurrencepb.EventRecurrence, error) {
	readReq := &eventrecurrencepb.ReadEventRecurrenceRequest{
		Data: &eventrecurrencepb.EventRecurrence{Id: eventRecurrenceID},
	}

	resp, err := uc.repositories.EventRecurrence.ReadEventRecurrence(ctx, readReq)
	if err != nil {
		// Check if this is a not found error from repository
		if contains := contextutil.Contains(err.Error(), "not found"); contains {
			errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": eventRecurrenceID}, "Event recurrence not found [DEFAULT]")
			return nil, errors.New(errorMessage)
		}
		return nil, err
	}

	if len(resp.Data) == 0 {
		errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": eventRecurrenceID}, "Event recurrence not found [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	return resp.Data[0], nil
}
