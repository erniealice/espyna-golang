package eventresource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ReadEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event resource operation
func (uc *ReadEventResourceUseCase) Execute(ctx context.Context, req *eventresourcepb.ReadEventResourceRequest) (*eventresourcepb.ReadEventResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventResource, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventResource, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_resource.errors.authorization_failed", "Authorization failed for event resource")
		return nil, errors.New(translatedError)
	}

	// Call repository
	return uc.repositories.EventResource.ReadEventResource(ctx, req)
}
