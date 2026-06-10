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

// ReadEventResourceRepositories groups all repository dependencies
type ReadEventResourceRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// ReadEventResourceServices groups all business service dependencies
type ReadEventResourceServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ReadEventResourceUseCase handles the business logic for reading event resource assignments
type ReadEventResourceUseCase struct {
	repositories ReadEventResourceRepositories
	services     ReadEventResourceServices
}

// NewReadEventResourceUseCase creates use case with grouped dependencies
func NewReadEventResourceUseCase(
	repositories ReadEventResourceRepositories,
	services ReadEventResourceServices,
) *ReadEventResourceUseCase {
	return &ReadEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventResourceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventResourceUseCase with grouped parameters instead
func NewReadEventResourceUseCaseUngrouped(
	eventResourceRepo eventresourcepb.EventResourceDomainServiceServer,
) *ReadEventResourceUseCase {
	repositories := ReadEventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         nil,
	}

	services := ReadEventResourceServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ReadEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event resource operation
func (uc *ReadEventResourceUseCase) Execute(ctx context.Context, req *eventresourcepb.ReadEventResourceRequest) (*eventresourcepb.ReadEventResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventResource, entityid.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventResource, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	// Call repository
	return uc.repositories.EventResource.ReadEventResource(ctx, req)
}
