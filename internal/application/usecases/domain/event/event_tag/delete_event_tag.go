package eventtag

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// DeleteEventTagRepositories groups all repository dependencies
type DeleteEventTagRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// DeleteEventTagServices groups all business service dependencies
type DeleteEventTagServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteEventTagUseCase handles the business logic for deleting an event_tag
type DeleteEventTagUseCase struct {
	repositories DeleteEventTagRepositories
	services     DeleteEventTagServices
}

// NewDeleteEventTagUseCase creates use case with grouped dependencies
func NewDeleteEventTagUseCase(
	repositories DeleteEventTagRepositories,
	services DeleteEventTagServices,
) *DeleteEventTagUseCase {
	return &DeleteEventTagUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event_tag operation
func (uc *DeleteEventTagUseCase) Execute(ctx context.Context, req *eventtagpb.DeleteEventTagRequest) (*eventtagpb.DeleteEventTagResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventTag,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventTag, entityid.ActionDelete)
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

	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	return uc.repositories.EventTag.DeleteEventTag(ctx, req)
}

func (uc *DeleteEventTagUseCase) validateInput(req *eventtagpb.DeleteEventTagRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event_tag data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event_tag ID is required")
	}
	return nil
}

func (uc *DeleteEventTagUseCase) validateBusinessRules(eventTag *eventtagpb.EventTag) error {
	// Reference-check (GetEventTagInUseIDs) is enforced at the list-page builder
	// layer via contrib/postgres/reference/checker.go before the delete action
	// is enabled. No additional validation here.
	return nil
}
