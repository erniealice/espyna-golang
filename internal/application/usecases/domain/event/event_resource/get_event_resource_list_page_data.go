package eventresource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// GetEventResourceListPageDataRepositories groups all repository dependencies
type GetEventResourceListPageDataRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// GetEventResourceListPageDataServices groups all business service dependencies
type GetEventResourceListPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// GetEventResourceListPageDataUseCase handles the business logic for getting event resource list page data
type GetEventResourceListPageDataUseCase struct {
	repositories GetEventResourceListPageDataRepositories
	services     GetEventResourceListPageDataServices
}

// NewGetEventResourceListPageDataUseCase creates a new GetEventResourceListPageDataUseCase
func NewGetEventResourceListPageDataUseCase(
	repositories GetEventResourceListPageDataRepositories,
	services GetEventResourceListPageDataServices,
) *GetEventResourceListPageDataUseCase {
	return &GetEventResourceListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event resource list page data operation
func (uc *GetEventResourceListPageDataUseCase) Execute(ctx context.Context, req *eventresourcepb.GetEventResourceListPageDataRequest) (*eventresourcepb.GetEventResourceListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityEventResource, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventResource, ports.ActionList)
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

	// Handle nil request by creating default empty request
	if req == nil {
		req = &eventresourcepb.GetEventResourceListPageDataRequest{}
	}

	// Call repository
	return uc.repositories.EventResource.GetEventResourceListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventResourceListPageDataUseCase) validateInput(req *eventresourcepb.GetEventResourceListPageDataRequest) error {
	// For list page data operations, nil request is allowed - we'll create a default empty request
	return nil
}
