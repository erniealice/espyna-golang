package eventresource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// ListEventResourcesRepositories groups all repository dependencies
type ListEventResourcesRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// ListEventResourcesServices groups all business service dependencies
type ListEventResourcesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ListEventResourcesUseCase handles the business logic for listing event resource assignments
type ListEventResourcesUseCase struct {
	repositories ListEventResourcesRepositories
	services     ListEventResourcesServices
}

// NewListEventResourcesUseCase creates a new ListEventResourcesUseCase
func NewListEventResourcesUseCase(
	repositories ListEventResourcesRepositories,
	services ListEventResourcesServices,
) *ListEventResourcesUseCase {
	return &ListEventResourcesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventResourcesUseCaseUngrouped creates a new ListEventResourcesUseCase
// Deprecated: Use NewListEventResourcesUseCase with grouped parameters instead
func NewListEventResourcesUseCaseUngrouped(eventResourceRepo eventresourcepb.EventResourceDomainServiceServer) *ListEventResourcesUseCase {
	repositories := ListEventResourcesRepositories{
		EventResource: eventResourceRepo,
		Event:         nil,
	}

	services := ListEventResourcesServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventResourcesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event resources operation
func (uc *ListEventResourcesUseCase) Execute(ctx context.Context, req *eventresourcepb.ListEventResourcesRequest) (*eventresourcepb.ListEventResourcesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventResource, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventResource, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Handle nil request by creating default empty request for list operations
	if req == nil {
		req = &eventresourcepb.ListEventResourcesRequest{}
	}

	// Call repository
	return uc.repositories.EventResource.ListEventResources(ctx, req)
}

// validateInput validates the input request
func (uc *ListEventResourcesUseCase) validateInput(req *eventresourcepb.ListEventResourcesRequest) error {
	// For list operations, nil request is allowed - we'll create a default empty request
	return nil
}
