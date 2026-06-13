package subscription_workspace_user

import (
	"context"
	"errors"
	"fmt"

	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// DeleteSubscriptionWorkspaceUserRepositories groups all repository dependencies
type DeleteSubscriptionWorkspaceUserRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// DeleteSubscriptionWorkspaceUserServices groups all business service dependencies
type DeleteSubscriptionWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteSubscriptionWorkspaceUserUseCase handles the business logic for deleting subscription workspace users (soft-delete)
type DeleteSubscriptionWorkspaceUserUseCase struct {
	repositories DeleteSubscriptionWorkspaceUserRepositories
	services     DeleteSubscriptionWorkspaceUserServices
}

// NewDeleteSubscriptionWorkspaceUserUseCase creates a new DeleteSubscriptionWorkspaceUserUseCase
func NewDeleteSubscriptionWorkspaceUserUseCase(
	repositories DeleteSubscriptionWorkspaceUserRepositories,
	services DeleteSubscriptionWorkspaceUserServices,
) *DeleteSubscriptionWorkspaceUserUseCase {
	return &DeleteSubscriptionWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete subscription workspace user operation (soft-delete: active=false).
func (uc *DeleteSubscriptionWorkspaceUserUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SubscriptionWorkspaceUser,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SubscriptionWorkspaceUser.DeleteSubscriptionWorkspaceUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.deletion_failed", "Subscription workspace user deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *DeleteSubscriptionWorkspaceUserUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserRequest) error {
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
