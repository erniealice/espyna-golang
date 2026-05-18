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

// ReadEventAttendeeRepositories groups all repository dependencies
type ReadEventAttendeeRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// ReadEventAttendeeServices groups all business service dependencies
type ReadEventAttendeeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadEventAttendeeUseCase handles the business logic for reading event attendee associations
type ReadEventAttendeeUseCase struct {
	repositories ReadEventAttendeeRepositories
	services     ReadEventAttendeeServices
}

// NewReadEventAttendeeUseCase creates use case with grouped dependencies
func NewReadEventAttendeeUseCase(
	repositories ReadEventAttendeeRepositories,
	services ReadEventAttendeeServices,
) *ReadEventAttendeeUseCase {
	return &ReadEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventAttendeeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventAttendeeUseCase with grouped parameters instead
func NewReadEventAttendeeUseCaseUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
) *ReadEventAttendeeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadEventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         nil,
	}

	services := ReadEventAttendeeServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ReadEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event attendee operation
func (uc *ReadEventAttendeeUseCase) Execute(ctx context.Context, req *eventattendeepb.ReadEventAttendeeRequest) (*eventattendeepb.ReadEventAttendeeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventAttendee, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attendee.validation.id_required", "[ERR-DEFAULT] ID is required"))
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

	// Call repository
	return uc.repositories.EventAttendee.ReadEventAttendee(ctx, req)
}
