package eventattendee

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// GetEventAttendeeListPageDataRepositories groups all repository dependencies
type GetEventAttendeeListPageDataRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// GetEventAttendeeListPageDataServices groups all business service dependencies
type GetEventAttendeeListPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// GetEventAttendeeListPageDataUseCase handles the business logic for getting event attendee list page data
type GetEventAttendeeListPageDataUseCase struct {
	repositories GetEventAttendeeListPageDataRepositories
	services     GetEventAttendeeListPageDataServices
}

// NewGetEventAttendeeListPageDataUseCase creates a new GetEventAttendeeListPageDataUseCase
func NewGetEventAttendeeListPageDataUseCase(
	repositories GetEventAttendeeListPageDataRepositories,
	services GetEventAttendeeListPageDataServices,
) *GetEventAttendeeListPageDataUseCase {
	return &GetEventAttendeeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event attendee list page data operation
func (uc *GetEventAttendeeListPageDataUseCase) Execute(ctx context.Context, req *eventattendeepb.GetEventAttendeeListPageDataRequest) (*eventattendeepb.GetEventAttendeeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventAttendee, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventAttendee, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Handle nil request by creating default empty request
	if req == nil {
		req = &eventattendeepb.GetEventAttendeeListPageDataRequest{}
	}

	// Call repository
	return uc.repositories.EventAttendee.GetEventAttendeeListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventAttendeeListPageDataUseCase) validateInput(req *eventattendeepb.GetEventAttendeeListPageDataRequest) error {
	// For list page data operations, nil request is allowed - we'll create a default empty request
	return nil
}
