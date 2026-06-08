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

// ListEventTagsRepositories groups all repository dependencies
type ListEventTagsRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// ListEventTagsServices groups all business service dependencies
type ListEventTagsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListEventTagsUseCase handles the business logic for listing event_tags
type ListEventTagsUseCase struct {
	repositories ListEventTagsRepositories
	services     ListEventTagsServices
}

// NewListEventTagsUseCase creates use case with grouped dependencies
func NewListEventTagsUseCase(
	repositories ListEventTagsRepositories,
	services ListEventTagsServices,
) *ListEventTagsUseCase {
	return &ListEventTagsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event_tags operation
func (uc *ListEventTagsUseCase) Execute(ctx context.Context, req *eventtagpb.ListEventTagsRequest) (*eventtagpb.ListEventTagsResponse, error) {
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
		req = &eventtagpb.ListEventTagsRequest{}
	}

	return uc.repositories.EventTag.ListEventTags(ctx, req)
}
