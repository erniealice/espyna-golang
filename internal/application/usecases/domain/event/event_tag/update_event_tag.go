package eventtag

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// UpdateEventTagRepositories groups all repository dependencies
type UpdateEventTagRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer
}

// UpdateEventTagServices groups all business service dependencies
type UpdateEventTagServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateEventTagUseCase handles the business logic for updating an event_tag
type UpdateEventTagUseCase struct {
	repositories UpdateEventTagRepositories
	services     UpdateEventTagServices
}

// NewUpdateEventTagUseCase creates use case with grouped dependencies
func NewUpdateEventTagUseCase(
	repositories UpdateEventTagRepositories,
	services UpdateEventTagServices,
) *UpdateEventTagUseCase {
	return &UpdateEventTagUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event_tag operation
func (uc *UpdateEventTagUseCase) Execute(ctx context.Context, req *eventtagpb.UpdateEventTagRequest) (*eventtagpb.UpdateEventTagResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventTag, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_tag.errors.authorization_failed", "Authorization failed for event_tag")
		return nil, errors.New(translatedError)
	}

	permission := entityid.Permission(entityid.EventTag, entityid.ActionUpdate)
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

	if err := uc.enrichEventTagData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	return uc.repositories.EventTag.UpdateEventTag(ctx, req)
}

func (uc *UpdateEventTagUseCase) validateInput(req *eventtagpb.UpdateEventTagRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event_tag data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event_tag ID is required")
	}
	if req.Data.Name == "" {
		return errors.New("event_tag name is required")
	}
	return nil
}

func (uc *UpdateEventTagUseCase) enrichEventTagData(eventTag *eventtagpb.EventTag) error {
	now := time.Now()
	eventTag.DateModified = &[]int64{now.UnixMilli()}[0]
	eventTag.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}

func (uc *UpdateEventTagUseCase) validateBusinessRules(eventTag *eventtagpb.EventTag) error {
	return nil
}
