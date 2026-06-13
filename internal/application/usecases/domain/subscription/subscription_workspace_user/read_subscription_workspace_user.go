package subscription_workspace_user

import (
	"context"
	"errors"

	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// ReadSubscriptionWorkspaceUserRepositories groups all repository dependencies
type ReadSubscriptionWorkspaceUserRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// ReadSubscriptionWorkspaceUserServices groups all business service dependencies
type ReadSubscriptionWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadSubscriptionWorkspaceUserUseCase handles the business logic for reading subscription workspace users
type ReadSubscriptionWorkspaceUserUseCase struct {
	repositories ReadSubscriptionWorkspaceUserRepositories
	services     ReadSubscriptionWorkspaceUserServices
}

// NewReadSubscriptionWorkspaceUserUseCase creates a new ReadSubscriptionWorkspaceUserUseCase
func NewReadSubscriptionWorkspaceUserUseCase(
	repositories ReadSubscriptionWorkspaceUserRepositories,
	services ReadSubscriptionWorkspaceUserServices,
) *ReadSubscriptionWorkspaceUserUseCase {
	return &ReadSubscriptionWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read subscription workspace user operation
func (uc *ReadSubscriptionWorkspaceUserUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionWorkspaceUser,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionWorkspaceUser.ReadSubscriptionWorkspaceUser(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadSubscriptionWorkspaceUserUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.validation.id_required", "Subscription workspace user ID is required [DEFAULT]"))
	}
	return nil
}
