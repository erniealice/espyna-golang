package eventclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"

	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	eventpb "leapfor.xyz/esqyma/golang/v1/domain/event/event"
	eventclientpb "leapfor.xyz/esqyma/golang/v1/domain/event/event_client"
)

// CreateEventClientRepositories groups all repository dependencies
type CreateEventClientRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// CreateEventClientServices groups all business service dependencies
type CreateEventClientServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEventClientUseCase handles the business logic for creating event client associations
type CreateEventClientUseCase struct {
	repositories CreateEventClientRepositories
	services     CreateEventClientServices
}

// NewCreateEventClientUseCase creates use case with grouped dependencies
func NewCreateEventClientUseCase(
	repositories CreateEventClientRepositories,
	services CreateEventClientServices,
) *CreateEventClientUseCase {
	return &CreateEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventClientUseCase with grouped parameters instead
func NewCreateEventClientUseCaseUngrouped(
	eventClientRepo eventclientpb.EventClientDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
) *CreateEventClientUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateEventClientRepositories{
		EventClient: eventClientRepo,
		Event:       eventRepo,
		Client:      clientRepo,
	}

	services := CreateEventClientServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return &CreateEventClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event client operation
func (uc *CreateEventClientUseCase) Execute(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, ports.ErrUserNotAuthenticated()
		}

		// TODO: Implement workspace-scoped permission check for event client creation
		// permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionCreate)
		// authorized, err := uc.services.AuthorizationService.HasPermissionInWorkspace(ctx, userID, workspaceID, permission)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichEventClientData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *CreateEventClientUseCase) shouldUseTransaction(ctx context.Context) bool {
	// Use transaction if:
	// 1. TransactionService is available, AND
	// 2. We're not already in a transaction context
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}

	// Don't start a nested transaction if we're already in one
	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreateEventClientUseCase) executeWithTransaction(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	var response *eventclientpb.CreateEventClientResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// All validations and operations within transaction

		// Business rule validation (check first to avoid unnecessary DB calls)
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			return err
		}

		// Create EventClient (will participate in transaction)
		createResponse, err := uc.repositories.EventClient.CreateEventClient(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event client: %w", err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction (backward compatibility)
func (uc *CreateEventClientUseCase) executeWithoutTransaction(ctx context.Context, req *eventclientpb.CreateEventClientRequest) (*eventclientpb.CreateEventClientResponse, error) {
	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	// Call repository (no transaction)
	return uc.repositories.EventClient.CreateEventClient(ctx, req)
}

// validateInput validates the input request
func (uc *CreateEventClientUseCase) validateInput(req *eventclientpb.CreateEventClientRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event client data is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ClientId == "" {
		return errors.New("client ID is required")
	}
	return nil
}

// enrichEventClientData adds generated fields and audit information
func (uc *CreateEventClientUseCase) enrichEventClientData(eventClient *eventclientpb.EventClient) error {
	now := time.Now()

	// Generate EventClient ID if not provided
	if eventClient.Id == "" {
		eventClient.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	eventClient.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventClient.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventClient.DateModified = &[]int64{now.UnixMilli()}[0]
	eventClient.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	eventClient.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateEventClientUseCase) validateBusinessRules(eventClient *eventclientpb.EventClient) error {
	// Validate event and client relationship
	if eventClient.EventId == eventClient.ClientId {
		return errors.New("event ID and client ID cannot be the same")
	}

	// Additional business rules can be added here
	// - Check if client can be associated with the event
	// - Validate event capacity
	// - Check for scheduling conflicts

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateEventClientUseCase) validateEntityReferences(ctx context.Context, eventClient *eventclientpb.EventClient) error {
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
