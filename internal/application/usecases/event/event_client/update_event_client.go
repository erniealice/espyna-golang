package eventclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// UpdateEventClientRepositories groups all repository dependencies
type UpdateEventClientRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// UpdateEventClientServices groups all business service dependencies
type UpdateEventClientServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateEventClientUseCase handles the business logic for updating event client associations
type UpdateEventClientUseCase struct {
	repositories UpdateEventClientRepositories
	services     UpdateEventClientServices
}

// NewUpdateEventClientUseCase creates a new UpdateEventClientUseCase
func NewUpdateEventClientUseCase(
	repositories UpdateEventClientRepositories,
	services UpdateEventClientServices,
) *UpdateEventClientUseCase {
	return &UpdateEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateEventClientUseCaseUngrouped creates a new UpdateEventClientUseCase
// Deprecated: Use NewUpdateEventClientUseCase with grouped parameters instead
func NewUpdateEventClientUseCaseUngrouped(eventClientRepo eventclientpb.EventClientDomainServiceServer, eventRepo eventpb.EventDomainServiceServer, clientRepo clientpb.ClientDomainServiceServer) *UpdateEventClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateEventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}

	services := UpdateEventClientServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &UpdateEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event client operation
func (uc *UpdateEventClientUseCase) Execute(ctx context.Context, req *eventclientpb.UpdateEventClientRequest) (*eventclientpb.UpdateEventClientResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichEventClientData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventClient.UpdateEventClient(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateEventClientUseCase) validateInput(req *eventclientpb.UpdateEventClientRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event client data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event client ID is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ClientId == "" {
		return errors.New("client ID is required")
	}
	return nil
}

// enrichEventClientData adds audit information for updates
func (uc *UpdateEventClientUseCase) enrichEventClientData(eventClient *eventclientpb.EventClient) error {
	now := time.Now()

	// Update audit fields
	eventClient.DateModified = &[]int64{now.Unix()}[0]
	eventClient.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateEventClientUseCase) validateBusinessRules(eventClient *eventclientpb.EventClient) error {
	// Validate that event and client IDs are not the same
	if eventClient.EventId == eventClient.ClientId {
		return errors.New("event ID and client ID cannot be the same")
	}

	// Additional business rules can be added here
	// - Check if client can still be associated with the event
	// - Validate updated event capacity
	// - Check for scheduling conflicts

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateEventClientUseCase) validateEntityReferences(ctx context.Context, eventClient *eventclientpb.EventClient) error {
	// Validate Event entity reference
	if eventClient.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventClient.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventClient.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventClient.EventId)
		}
	}

	// Validate Client entity reference
	if eventClient.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: eventClient.ClientId},
		})
		if err != nil {
			return err
		}
		if client == nil || client.Data == nil || len(client.Data) == 0 {
			return fmt.Errorf("referenced client with ID '%s' does not exist", eventClient.ClientId)
		}
		if !client.Data[0].Active {
			return fmt.Errorf("referenced client with ID '%s' is not active", eventClient.ClientId)
		}
	}

	return nil
}
