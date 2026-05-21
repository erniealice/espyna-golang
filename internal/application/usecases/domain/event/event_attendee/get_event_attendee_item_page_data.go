package eventattendee

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// GetEventAttendeeItemPageDataRepositories groups all repository dependencies
type GetEventAttendeeItemPageDataRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// GetEventAttendeeItemPageDataServices groups all business service dependencies
type GetEventAttendeeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// GetEventAttendeeItemPageDataUseCase handles the business logic for getting event attendee item page data
type GetEventAttendeeItemPageDataUseCase struct {
	repositories GetEventAttendeeItemPageDataRepositories
	services     GetEventAttendeeItemPageDataServices
}

// NewGetEventAttendeeItemPageDataUseCase creates a new GetEventAttendeeItemPageDataUseCase
func NewGetEventAttendeeItemPageDataUseCase(
	repositories GetEventAttendeeItemPageDataRepositories,
	services GetEventAttendeeItemPageDataServices,
) *GetEventAttendeeItemPageDataUseCase {
	return &GetEventAttendeeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event attendee item page data operation
func (uc *GetEventAttendeeItemPageDataUseCase) Execute(ctx context.Context, req *eventattendeepb.GetEventAttendeeItemPageDataRequest) (*eventattendeepb.GetEventAttendeeItemPageDataResponse, error) {
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

	permission := ports.EntityPermission(ports.EntityEventAttendee, ports.ActionRead)
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

	// Call repository
	return uc.repositories.EventAttendee.GetEventAttendeeItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventAttendeeItemPageDataUseCase) validateInput(req *eventattendeepb.GetEventAttendeeItemPageDataRequest) error {
	if req == nil {
		return errors.New("Request cannot be nil")
	}

	if req.EventAttendeeId == "" {
		return errors.New("Event attendee ID is required")
	}

	return nil
}
