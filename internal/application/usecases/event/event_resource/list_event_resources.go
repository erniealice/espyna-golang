package eventresource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil, // Will be injected later if needed
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ListEventResourcesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event resources operation
func (uc *ListEventResourcesUseCase) Execute(ctx context.Context, req *eventresourcepb.ListEventResourcesRequest) (*eventresourcepb.ListEventResourcesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventResource, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventResource, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
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
