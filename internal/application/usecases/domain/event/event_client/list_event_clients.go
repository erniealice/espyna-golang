package eventclient

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
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
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventClientsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event clients operation
func (uc *ListEventClientsUseCase) Execute(ctx context.Context, req *eventclientpb.ListEventClientsRequest) (*eventclientpb.ListEventClientsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventClient,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventClient, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
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
