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

// GetEventClientListPageDataRepositories groups all repository dependencies
type GetEventClientListPageDataRepositories struct {
	EventClient eventclientpb.EventClientDomainServiceServer // Primary entity repository
	Event       eventpb.EventDomainServiceServer             // Entity reference validation
	Client      clientpb.ClientDomainServiceServer           // Entity reference validation
}

// GetEventClientListPageDataServices groups all business service dependencies
type GetEventClientListPageDataServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// GetEventClientListPageDataUseCase handles the business logic for getting event client list page data
type GetEventClientListPageDataUseCase struct {
	repositories GetEventClientListPageDataRepositories
	services     GetEventClientListPageDataServices
}

// NewGetEventClientListPageDataUseCase creates a new GetEventClientListPageDataUseCase
func NewGetEventClientListPageDataUseCase(
	repositories GetEventClientListPageDataRepositories,
	services GetEventClientListPageDataServices,
) *GetEventClientListPageDataUseCase {
	return &GetEventClientListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event client list page data operation
func (uc *GetEventClientListPageDataUseCase) Execute(ctx context.Context, req *eventclientpb.GetEventClientListPageDataRequest) (*eventclientpb.GetEventClientListPageDataResponse, error) {
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

	permission := entityid.EntityPermission(entityid.EventClient, entityid.ActionList)
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

	// Handle nil request by creating default empty request
	if req == nil {
		req = &eventclientpb.GetEventClientListPageDataRequest{}
	}

	// Call repository
	return uc.repositories.EventClient.GetEventClientListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetEventClientListPageDataUseCase) validateInput(req *eventclientpb.GetEventClientListPageDataRequest) error {
	// For list page data operations, nil request is allowed - we'll create a default empty request
	return nil
}
