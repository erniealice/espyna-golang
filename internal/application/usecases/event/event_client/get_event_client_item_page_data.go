package eventclient

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// GetEventClientItemPageDataRepositories groups all repository dependencies
type GetEventClientItemPageDataRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// GetEventClientItemPageDataServices groups all business service dependencies
type GetEventClientItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// GetEventClientItemPageDataUseCase handles the business logic for getting event client item page data
type GetEventClientItemPageDataUseCase struct {
	repositories GetEventClientItemPageDataRepositories
	services     GetEventClientItemPageDataServices
}

// NewGetEventClientItemPageDataUseCase creates a new GetEventClientItemPageDataUseCase
func NewGetEventClientItemPageDataUseCase(
	repositories GetEventClientItemPageDataRepositories,
	services GetEventClientItemPageDataServices,
) *GetEventClientItemPageDataUseCase {
	return &GetEventClientItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event client item page data operation
func (uc *GetEventClientItemPageDataUseCase) Execute(ctx context.Context, req *eventclientpb.GetEventClientItemPageDataRequest) (*eventclientpb.GetEventClientItemPageDataResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventClient.GetEventClientItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventClientItemPageDataUseCase) validateInput(req *eventclientpb.GetEventClientItemPageDataRequest) error {
	if req == nil {
		translatedError := "Request cannot be nil"
		return errors.New(translatedError)
	}

	if req.EventClientId == "" {
		translatedError := "Event client ID is required"
		return errors.New(translatedError)
	}

	return nil
}
