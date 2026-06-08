package eventtag

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// GetEventTagListPageDataRepositories groups all repository dependencies
type GetEventTagListPageDataRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// GetEventTagListPageDataServices groups all business service dependencies
type GetEventTagListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetEventTagListPageDataUseCase handles the business logic for getting event_tag list page data
type GetEventTagListPageDataUseCase struct {
	repositories GetEventTagListPageDataRepositories
	services     GetEventTagListPageDataServices
}

// NewGetEventTagListPageDataUseCase creates use case with grouped dependencies
func NewGetEventTagListPageDataUseCase(
	repositories GetEventTagListPageDataRepositories,
	services GetEventTagListPageDataServices,
) *GetEventTagListPageDataUseCase {
	return &GetEventTagListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event_tag list page data operation
func (uc *GetEventTagListPageDataUseCase) Execute(ctx context.Context, req *eventtagpb.GetEventTagListPageDataRequest) (*eventtagpb.GetEventTagListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventTag, entityid.ActionList); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	permission := entityid.Permission(entityid.EventTag, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	if req == nil {
		req = &eventtagpb.GetEventTagListPageDataRequest{}
	}

	return uc.repositories.EventTag.GetEventTagListPageData(ctx, req)
}
