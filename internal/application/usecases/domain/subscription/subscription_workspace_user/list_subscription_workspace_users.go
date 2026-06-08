package subscription_workspace_user

import (
	"context"
	"errors"

	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ListSubscriptionWorkspaceUsersRepositories groups all repository dependencies
type ListSubscriptionWorkspaceUsersRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// ListSubscriptionWorkspaceUsersServices groups all business service dependencies
type ListSubscriptionWorkspaceUsersServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListSubscriptionWorkspaceUsersUseCase handles the business logic for listing subscription workspace users
type ListSubscriptionWorkspaceUsersUseCase struct {
	repositories ListSubscriptionWorkspaceUsersRepositories
	services     ListSubscriptionWorkspaceUsersServices
}

// NewListSubscriptionWorkspaceUsersUseCase creates a new ListSubscriptionWorkspaceUsersUseCase
func NewListSubscriptionWorkspaceUsersUseCase(
	repositories ListSubscriptionWorkspaceUsersRepositories,
	services ListSubscriptionWorkspaceUsersServices,
) *ListSubscriptionWorkspaceUsersUseCase {
	return &ListSubscriptionWorkspaceUsersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list subscription workspace users operation. Filters by
// subscription_id and by workspace_user_id (the "what do I service" query) ride
// on req.Filters.
func (uc *ListSubscriptionWorkspaceUsersUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersRequest) (*subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionWorkspaceUser, entityid.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionWorkspaceUser.ListSubscriptionWorkspaceUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ListSubscriptionWorkspaceUsersUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
