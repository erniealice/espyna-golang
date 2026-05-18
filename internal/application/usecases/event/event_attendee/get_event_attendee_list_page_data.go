package eventattendee

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventAttendee, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttendee, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.errors.authorization_failed", "Authorization failed for event attendee")
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
