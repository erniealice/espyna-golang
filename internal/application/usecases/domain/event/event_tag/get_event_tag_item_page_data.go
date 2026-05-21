package eventtag

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// GetEventTagItemPageDataRepositories groups all repository dependencies
type GetEventTagItemPageDataRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// GetEventTagItemPageDataServices groups all business service dependencies
type GetEventTagItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetEventTagItemPageDataUseCase handles the business logic for getting event_tag item page data
type GetEventTagItemPageDataUseCase struct {
	repositories GetEventTagItemPageDataRepositories
	services     GetEventTagItemPageDataServices
}

// NewGetEventTagItemPageDataUseCase creates use case with grouped dependencies
func NewGetEventTagItemPageDataUseCase(
	repositories GetEventTagItemPageDataRepositories,
	services GetEventTagItemPageDataServices,
) *GetEventTagItemPageDataUseCase {
	return &GetEventTagItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event_tag item page data operation
func (uc *GetEventTagItemPageDataUseCase) Execute(ctx context.Context, req *eventtagpb.GetEventTagItemPageDataRequest) (*eventtagpb.GetEventTagItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityEventTag, ports.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventTag, ports.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	return uc.repositories.EventTag.GetEventTagItemPageData(ctx, req)
}

func (uc *GetEventTagItemPageDataUseCase) validateInput(req *eventtagpb.GetEventTagItemPageDataRequest) error {
	if req == nil {
		return errors.New("Request cannot be nil")
	}
	if req.EventTagId == "" {
		return errors.New("event_tag ID is required")
	}
	return nil
}
