package eventrecurrence

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// ReadEventRecurrenceRepositories groups all repository dependencies
type ReadEventRecurrenceRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// ReadEventRecurrenceServices groups all business service dependencies
type ReadEventRecurrenceServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadEventRecurrenceUseCase handles the business logic for reading a single event recurrence
type ReadEventRecurrenceUseCase struct {
	repositories ReadEventRecurrenceRepositories
	services     ReadEventRecurrenceServices
}

// NewReadEventRecurrenceUseCase creates use case with grouped dependencies
func NewReadEventRecurrenceUseCase(
	repositories ReadEventRecurrenceRepositories,
	services ReadEventRecurrenceServices,
) *ReadEventRecurrenceUseCase {
	return &ReadEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventRecurrenceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventRecurrenceUseCase with grouped parameters instead
func NewReadEventRecurrenceUseCaseUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer) *ReadEventRecurrenceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadEventRecurrenceRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := ReadEventRecurrenceServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ReadEventRecurrenceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event recurrence operation
func (uc *ReadEventRecurrenceUseCase) Execute(ctx context.Context, req *eventrecurrencepb.ReadEventRecurrenceRequest) (*eventrecurrencepb.ReadEventRecurrenceResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "event_recurrence",
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.EventRecurrence.ReadEventRecurrence(ctx, req)
	if err != nil {
		// Check if this is a not found error from repository
		if contains := contextutil.Contains(err.Error(), "not found"); contains {
			errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": req.Data.Id}, "Event recurrence not found [DEFAULT]")
			return nil, errors.New(errorMessage)
		}
		return nil, err
	}

	// Business logic validation
	if len(resp.Data) == 0 {
		errorMessage := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "event_recurrence.errors.not_found", map[string]interface{}{"eventRecurrenceId": req.Data.Id}, "Event recurrence not found [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadEventRecurrenceUseCase) validateInput(ctx context.Context, req *eventrecurrencepb.ReadEventRecurrenceRequest) error {
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
