package eventoccurrence

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
)

// GetEventOccurrenceListPageDataRepositories groups all repository dependencies
type GetEventOccurrenceListPageDataRepositories struct {
	EventOccurrence eventoccurrencepb.EventOccurrenceDomainServiceServer // Primary entity repository
}

// GetEventOccurrenceListPageDataServices groups all business service dependencies
type GetEventOccurrenceListPageDataServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetEventOccurrenceListPageDataUseCase handles the business logic for getting event occurrence list page data
type GetEventOccurrenceListPageDataUseCase struct {
	repositories GetEventOccurrenceListPageDataRepositories
	services     GetEventOccurrenceListPageDataServices
}

// NewGetEventOccurrenceListPageDataUseCase creates a new GetEventOccurrenceListPageDataUseCase
func NewGetEventOccurrenceListPageDataUseCase(
	repositories GetEventOccurrenceListPageDataRepositories,
	services GetEventOccurrenceListPageDataServices,
) *GetEventOccurrenceListPageDataUseCase {
	return &GetEventOccurrenceListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event occurrence list page data operation
func (uc *GetEventOccurrenceListPageDataUseCase) Execute(ctx context.Context, req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest) (*eventoccurrencepb.GetEventOccurrenceListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventOccurrence, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Handle nil request by creating default empty request
	if req == nil {
		req = &eventoccurrencepb.GetEventOccurrenceListPageDataRequest{}
	}

	// Call repository
	return uc.repositories.EventOccurrence.GetEventOccurrenceListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventOccurrenceListPageDataUseCase) validateInput(req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest) error {
	// For list page data operations, nil request is allowed — we'll create a default empty request
	if req != nil && req.Search != nil && req.Search.Query == "" {
		return errors.New("search query cannot be empty when search request is provided")
	}
	return nil
}

// ExecuteForCalendarRange performs the list page data query scoped to a time window.
// Callers pass start/end Unix timestamps via filter fields; this method delegates to the
// underlying repository which uses an optimised range index on (workspace_id, start_date_time_utc).
func (uc *GetEventOccurrenceListPageDataUseCase) ExecuteForCalendarRange(
	ctx context.Context,
	req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest,
) (*eventoccurrencepb.GetEventOccurrenceListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventOccurrence, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		errorMessage := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_occurrence.validation.request_required", "request is required")
		return nil, errors.New(errorMessage)
	}

	return uc.repositories.EventOccurrence.GetEventOccurrenceListPageData(ctx, req)
}
