package eventrecurrence

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventrecurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
)

// GetEventRecurrenceListPageDataRepositories groups all repository dependencies
type GetEventRecurrenceListPageDataRepositories struct {
	EventRecurrence eventrecurrencepb.EventRecurrenceDomainServiceServer // Primary entity repository
}

// GetEventRecurrenceListPageDataServices groups all business service dependencies
type GetEventRecurrenceListPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetEventRecurrenceListPageDataUseCase handles the business logic for getting event recurrence list page data
type GetEventRecurrenceListPageDataUseCase struct {
	repositories GetEventRecurrenceListPageDataRepositories
	services     GetEventRecurrenceListPageDataServices
}

// NewGetEventRecurrenceListPageDataUseCase creates a new GetEventRecurrenceListPageDataUseCase
func NewGetEventRecurrenceListPageDataUseCase(
	repositories GetEventRecurrenceListPageDataRepositories,
	services GetEventRecurrenceListPageDataServices,
) *GetEventRecurrenceListPageDataUseCase {
	return &GetEventRecurrenceListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event recurrence list page data operation
func (uc *GetEventRecurrenceListPageDataUseCase) Execute(
	ctx context.Context,
	req *eventrecurrencepb.GetEventRecurrenceListPageDataRequest,
) (*eventrecurrencepb.GetEventRecurrenceListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"event_recurrence", entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Handle nil request by creating default empty request
	if req == nil {
		req = &eventrecurrencepb.GetEventRecurrenceListPageDataRequest{}
	}

	// Call repository
	return uc.repositories.EventRecurrence.GetEventRecurrenceListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventRecurrenceListPageDataUseCase) validateInput(req *eventrecurrencepb.GetEventRecurrenceListPageDataRequest) error {
	// For list page data operations, nil request is allowed — we'll create a default empty request
	if req != nil && req.Search != nil && req.Search.Query == "" {
		return errors.New("search query cannot be empty when search request is provided")
	}
	return nil
}
