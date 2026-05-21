package eventtag

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// ReadEventTagRepositories groups all repository dependencies
type ReadEventTagRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// ReadEventTagServices groups all business service dependencies
type ReadEventTagServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadEventTagUseCase handles the business logic for reading an event_tag
type ReadEventTagUseCase struct {
	repositories ReadEventTagRepositories
	services     ReadEventTagServices
}

// NewReadEventTagUseCase creates use case with grouped dependencies
func NewReadEventTagUseCase(
	repositories ReadEventTagRepositories,
	services ReadEventTagServices,
) *ReadEventTagUseCase {
	return &ReadEventTagUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event_tag operation
func (uc *ReadEventTagUseCase) Execute(ctx context.Context, req *eventtagpb.ReadEventTagRequest) (*eventtagpb.ReadEventTagResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTag, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventTag, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	return uc.repositories.EventTag.ReadEventTag(ctx, req)
}
