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

// DeleteEventResourceRepositories groups all repository dependencies
type DeleteEventResourceRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// DeleteEventResourceServices groups all business service dependencies
type DeleteEventResourceServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// DeleteEventResourceUseCase handles the business logic for deleting event resource assignments
type DeleteEventResourceUseCase struct {
	repositories DeleteEventResourceRepositories
	services     DeleteEventResourceServices
}

// NewDeleteEventResourceUseCase creates a new DeleteEventResourceUseCase
func NewDeleteEventResourceUseCase(
	repositories DeleteEventResourceRepositories,
	services DeleteEventResourceServices,
) *DeleteEventResourceUseCase {
	return &DeleteEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteEventResourceUseCaseUngrouped creates a new DeleteEventResourceUseCase
// Deprecated: Use NewDeleteEventResourceUseCase with grouped parameters instead
func NewDeleteEventResourceUseCaseUngrouped(eventResourceRepo eventresourcepb.EventResourceDomainServiceServer) *DeleteEventResourceUseCase {
	repositories := DeleteEventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         nil,
	}

	services := DeleteEventResourceServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &DeleteEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event resource operation
func (uc *DeleteEventResourceUseCase) Execute(ctx context.Context, req *eventresourcepb.DeleteEventResourceRequest) (*eventresourcepb.DeleteEventResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityEventResource, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventResource, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventResource.DeleteEventResource(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteEventResourceUseCase) validateInput(req *eventresourcepb.DeleteEventResourceRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event resource data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event resource ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteEventResourceUseCase) validateBusinessRules(eventResource *eventresourcepb.EventResource) error {
	// Additional business rules can be added here
	// - Check if resource assignment can be safely released
	// - Validate impact on event resource capacity
	// - Check for related records that might be affected

	return nil
}
