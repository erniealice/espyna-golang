package eventrecurrence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// UpdateEventRecurrenceRepositories groups all repository dependencies
type UpdateEventRecurrenceRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// UpdateEventRecurrenceServices groups all business service dependencies
type UpdateEventRecurrenceServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateEventRecurrenceUseCase handles the business logic for updating event recurrences
type UpdateEventRecurrenceUseCase struct {
	repositories UpdateEventRecurrenceRepositories
	services     UpdateEventRecurrenceServices
}

// NewUpdateEventRecurrenceUseCase creates use case with grouped dependencies
func NewUpdateEventRecurrenceUseCase(
	repositories UpdateEventRecurrenceRepositories,
	services UpdateEventRecurrenceServices,
) *UpdateEventRecurrenceUseCase {
	return &UpdateEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateEventRecurrenceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateEventRecurrenceUseCase with grouped parameters instead
func NewUpdateEventRecurrenceUseCaseUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer) *UpdateEventRecurrenceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateEventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := UpdateEventRecurrenceServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return &UpdateEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event recurrence operation
func (uc *UpdateEventRecurrenceUseCase) Execute(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "event_recurrence",
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event recurrence update within a transaction
func (uc *UpdateEventRecurrenceUseCase) executeWithTransaction(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	var result *eventrecurrencepb.UpdateEventRecurrenceResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "event_recurrence.errors.update_failed", "Event recurrence update failed [DEFAULT]")
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
func (uc *UpdateEventRecurrenceUseCase) executeCore(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) (*eventrecurrencepb.UpdateEventRecurrenceResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Validate basic field requirements
	if err := uc.validateBasicFields(ctx, req.Data); err != nil {
		return nil, err
	}

	// Check if event recurrence exists
	_, err := uc.getExistingEventRecurrence(ctx, req.Data.Id)
	if err != nil {
		return nil, err
	}

	// Update audit fields
	uc.updateAuditFields(req.Data)

	// Call repository
	return uc.repositories.EventRecurrence.UpdateEventRecurrence(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateEventRecurrenceUseCase) validateInput(ctx context.Context, req *eventrecurrencepb.UpdateEventRecurrenceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.validation.data_required", "Event recurrence data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.validation.id_required", "Event recurrence ID is required [DEFAULT]"))
	}
	return nil
}

// validateBasicFields validates basic field requirements before business logic
func (uc *UpdateEventRecurrenceUseCase) validateBasicFields(ctx context.Context, eventRecurrence *eventrecurrencepb.EventRecurrence) error {
	if eventRecurrence.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.validation.name_required", "Event recurrence name is required [DEFAULT]"))
	}
	if eventRecurrence.RruleString == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.validation.rrule_string_required", "RRULE string is required [DEFAULT]"))
	}
	return nil
}

// getExistingEventRecurrence retrieves the current event recurrence state
func (uc *UpdateEventRecurrenceUseCase) getExistingEventRecurrence(ctx context.Context, eventRecurrenceID string) (*eventrecurrencepb.EventRecurrence, error) {
	readReq := &eventrecurrencepb.ReadEventRecurrenceRequest{
		Data: &eventrecurrencepb.EventRecurrence{Id: eventRecurrenceID},
	}

	resp, err := uc.repositories.EventRecurrence.ReadEventRecurrence(ctx, readReq)
	if err != nil {
		// Check if this is a not found error from repository
		if contains := contextutil.Contains(err.Error(), "not found"); contains {
			errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": eventRecurrenceID}, "Event recurrence not found [DEFAULT]")
			return nil, errors.New(errorMessage)
		}
		return nil, err
	}

	if len(resp.Data) == 0 {
		errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": eventRecurrenceID}, "Event recurrence not found [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	return resp.Data[0], nil
}

// updateAuditFields updates modification timestamps
func (uc *UpdateEventRecurrenceUseCase) updateAuditFields(eventRecurrence *eventrecurrencepb.EventRecurrence) {
	now := time.Now()
	ts := now.UnixMilli()
	tsStr := now.Format(time.RFC3339)
	eventRecurrence.DateModified = &ts
	eventRecurrence.DateModifiedString = &tsStr
}
