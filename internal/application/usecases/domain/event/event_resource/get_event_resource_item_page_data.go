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

// GetEventResourceItemPageDataRepositories groups all repository dependencies
type GetEventResourceItemPageDataRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// GetEventResourceItemPageDataServices groups all business service dependencies
type GetEventResourceItemPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// GetEventResourceItemPageDataUseCase handles the business logic for getting event resource item page data
type GetEventResourceItemPageDataUseCase struct {
	repositories GetEventResourceItemPageDataRepositories
	services     GetEventResourceItemPageDataServices
}

// NewGetEventResourceItemPageDataUseCase creates a new GetEventResourceItemPageDataUseCase
func NewGetEventResourceItemPageDataUseCase(
	repositories GetEventResourceItemPageDataRepositories,
	services GetEventResourceItemPageDataServices,
) *GetEventResourceItemPageDataUseCase {
	return &GetEventResourceItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event resource item page data operation
func (uc *GetEventResourceItemPageDataUseCase) Execute(ctx context.Context, req *eventresourcepb.GetEventResourceItemPageDataRequest) (*eventresourcepb.GetEventResourceItemPageDataResponse, error) {
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

	permission := ports.EntityPermission(ports.EntityEventResource, ports.ActionRead)
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

	// Call repository
	return uc.repositories.EventResource.GetEventResourceItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventResourceItemPageDataUseCase) validateInput(req *eventresourcepb.GetEventResourceItemPageDataRequest) error {
	if req == nil {
		return errors.New("Request cannot be nil")
	}

	if req.EventResourceId == "" {
		return errors.New("Event resource ID is required")
	}

	return nil
}
