package eventclient

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_client"
)

// GetEventClientItemPageDataRepositories groups all repository dependencies
type GetEventClientItemPageDataRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// GetEventClientItemPageDataServices groups all business service dependencies
type GetEventClientItemPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// GetEventClientItemPageDataUseCase handles the business logic for getting event client item page data
type GetEventClientItemPageDataUseCase struct {
	repositories GetEventClientItemPageDataRepositories
	services     GetEventClientItemPageDataServices
}

// NewGetEventClientItemPageDataUseCase creates a new GetEventClientItemPageDataUseCase
func NewGetEventClientItemPageDataUseCase(
	repositories GetEventClientItemPageDataRepositories,
	services GetEventClientItemPageDataServices,
) *GetEventClientItemPageDataUseCase {
	return &GetEventClientItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event client item page data operation
func (uc *GetEventClientItemPageDataUseCase) Execute(ctx context.Context, req *eventclientpb.GetEventClientItemPageDataRequest) (*eventclientpb.GetEventClientItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventClient, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventClient, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_client.errors.authorization_failed", "Authorization failed for schedule enrollment")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventClient.GetEventClientItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventClientItemPageDataUseCase) validateInput(req *eventclientpb.GetEventClientItemPageDataRequest) error {
	if req == nil {
		translatedError := "Request cannot be nil"
		return errors.New(translatedError)
	}

	if req.EventClientId == "" {
		translatedError := "Event client ID is required"
		return errors.New(translatedError)
	}

	return nil
}
