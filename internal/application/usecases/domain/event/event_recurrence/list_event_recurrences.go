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

// ListEventRecurrencesRepositories groups all repository dependencies
type ListEventRecurrencesRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// ListEventRecurrencesServices groups all business service dependencies
type ListEventRecurrencesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListEventRecurrencesUseCase handles the business logic for listing event recurrences
type ListEventRecurrencesUseCase struct {
	repositories ListEventRecurrencesRepositories
	services     ListEventRecurrencesServices
}

// NewListEventRecurrencesUseCase creates use case with grouped dependencies
func NewListEventRecurrencesUseCase(
	repositories ListEventRecurrencesRepositories,
	services ListEventRecurrencesServices,
) *ListEventRecurrencesUseCase {
	return &ListEventRecurrencesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventRecurrencesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListEventRecurrencesUseCase with grouped parameters instead
func NewListEventRecurrencesUseCaseUngrouped(eventRecurrenceRepo eventrecurrencepb.EventRecurrenceDomainServiceServer) *ListEventRecurrencesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListEventRecurrencesRepositories{
		EventRecurrence: eventRecurrenceRepo,
	}

	services := ListEventRecurrencesServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventRecurrencesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event recurrences operation
func (uc *ListEventRecurrencesUseCase) Execute(ctx context.Context, req *eventrecurrencepb.ListEventRecurrencesRequest) (*eventrecurrencepb.ListEventRecurrencesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "event_recurrence",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &eventrecurrencepb.ListEventRecurrencesRequest{}
	}

	// Call repository
	resp, err := uc.repositories.EventRecurrence.ListEventRecurrences(ctx, req)
	if err != nil {
		errorMessage := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_recurrence.errors.list_failed", "Failed to retrieve event recurrences [DEFAULT]")
		return nil, errors.New(errorMessage)
	}

	// Business logic post-processing (if needed)
	// Currently no additional business rules for list operation

	return resp, nil
}
