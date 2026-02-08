package eventclient

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// DeleteEventClientRepositories groups all repository dependencies
type DeleteEventClientRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// DeleteEventClientServices groups all business service dependencies
type DeleteEventClientServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteEventClientUseCase handles the business logic for deleting event client associations
type DeleteEventClientUseCase struct {
	repositories DeleteEventClientRepositories
	services     DeleteEventClientServices
}

// NewDeleteEventClientUseCase creates a new DeleteEventClientUseCase
func NewDeleteEventClientUseCase(
	repositories DeleteEventClientRepositories,
	services DeleteEventClientServices,
) *DeleteEventClientUseCase {
	return &DeleteEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteEventClientUseCaseUngrouped creates a new DeleteEventClientUseCase
// Deprecated: Use NewDeleteEventClientUseCase with grouped parameters instead
func NewDeleteEventClientUseCaseUngrouped(eventClientRepo eventclientpb.EventClientDomainServiceServer) *DeleteEventClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteEventClientRepositories{
		EventClient: eventClientRepo,
		Event:       nil,
		Client:      nil,
	}

	services := DeleteEventClientServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &DeleteEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event client operation
func (uc *DeleteEventClientUseCase) Execute(ctx context.Context, req *eventclientpb.DeleteEventClientRequest) (*eventclientpb.DeleteEventClientResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventClient.DeleteEventClient(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteEventClientUseCase) validateInput(req *eventclientpb.DeleteEventClientRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event client data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event client ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteEventClientUseCase) validateBusinessRules(eventClient *eventclientpb.EventClient) error {
	// Additional business rules can be added here
	// - Check if event client association can be safely deleted
	// - Validate impact on event capacity
	// - Check for related records that might be affected

	return nil
}
