package subscription_workspace_user

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
)

// GetSubscriptionWorkspaceUserListPageDataRepositories groups all repository dependencies
type GetSubscriptionWorkspaceUserListPageDataRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// GetSubscriptionWorkspaceUserListPageDataServices groups all business service dependencies
type GetSubscriptionWorkspaceUserListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetSubscriptionWorkspaceUserListPageDataUseCase handles the business logic for getting subscription workspace user list page data
type GetSubscriptionWorkspaceUserListPageDataUseCase struct {
	repositories GetSubscriptionWorkspaceUserListPageDataRepositories
	services     GetSubscriptionWorkspaceUserListPageDataServices
}

// NewGetSubscriptionWorkspaceUserListPageDataUseCase creates a new GetSubscriptionWorkspaceUserListPageDataUseCase
func NewGetSubscriptionWorkspaceUserListPageDataUseCase(
	repositories GetSubscriptionWorkspaceUserListPageDataRepositories,
	services GetSubscriptionWorkspaceUserListPageDataServices,
) *GetSubscriptionWorkspaceUserListPageDataUseCase {
	return &GetSubscriptionWorkspaceUserListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription workspace user list page data operation
func (uc *GetSubscriptionWorkspaceUserListPageDataUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataRequest) (*subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionWorkspaceUser,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionWorkspaceUser.GetSubscriptionWorkspaceUserListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *GetSubscriptionWorkspaceUserListPageDataUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
