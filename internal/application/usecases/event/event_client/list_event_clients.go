package eventclient

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// ListEventClientsRepositories groups all repository dependencies
type ListEventClientsRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// ListEventClientsServices groups all business service dependencies
type ListEventClientsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListEventClientsUseCase handles the business logic for listing event client associations
type ListEventClientsUseCase struct {
	repositories ListEventClientsRepositories
	services     ListEventClientsServices
}

// NewListEventClientsUseCase creates a new ListEventClientsUseCase
func NewListEventClientsUseCase(
	repositories ListEventClientsRepositories,
	services ListEventClientsServices,
) *ListEventClientsUseCase {
	return &ListEventClientsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventClientsUseCaseUngrouped creates a new ListEventClientsUseCase
// Deprecated: Use NewListEventClientsUseCase with grouped parameters instead
func NewListEventClientsUseCaseUngrouped(eventClientRepo eventclientpb.EventClientDomainServiceServer) *ListEventClientsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListEventClientsRepositories{
		EventClient: eventClientRepo,
		Event:       nil,
		Client:      nil,
	}

	services := ListEventClientsServices{
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ListEventClientsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event clients operation
func (uc *ListEventClientsUseCase) Execute(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventClient, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventClient, ports.ActionList)
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

	// Handle nil request by creating default empty request for list operations
	if req == nil {
		req = &eventclientpb.ListEventClientsRequest{}
	}

	// Call repository
	return uc.repositories.EventClient.ListEventClients(ctx, req)
}

// validateInput validates the input request
func (uc *ListEventClientsUseCase) validateInput(req *eventclientpb.ListEventClientsRequest) error {
	// For list operations, nil request is allowed - we'll create a default empty request
	return nil
}
