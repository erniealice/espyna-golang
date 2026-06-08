package subscription_workspace_user

import (
	"context"
	"errors"
	"fmt"
	"time"

	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// UpdateSubscriptionWorkspaceUserRepositories groups all repository dependencies
type UpdateSubscriptionWorkspaceUserRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
}

// UpdateSubscriptionWorkspaceUserServices groups all business service dependencies
type UpdateSubscriptionWorkspaceUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateSubscriptionWorkspaceUserUseCase handles the business logic for updating subscription workspace users
type UpdateSubscriptionWorkspaceUserUseCase struct {
	repositories UpdateSubscriptionWorkspaceUserRepositories
	services     UpdateSubscriptionWorkspaceUserServices
}

// NewUpdateSubscriptionWorkspaceUserUseCase creates a new UpdateSubscriptionWorkspaceUserUseCase
func NewUpdateSubscriptionWorkspaceUserUseCase(
	repositories UpdateSubscriptionWorkspaceUserRepositories,
	services UpdateSubscriptionWorkspaceUserServices,
) *UpdateSubscriptionWorkspaceUserUseCase {
	return &UpdateSubscriptionWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update subscription workspace user operation.
//
// client_id and subscription_id are NEVER reassigned here. They form the
// composite-FK identity (subscription_id FK + (client_id, workspace_user_id) FK)
// and are stamped only at create from the parent subscription's client_id
// (single-write boundary). If the generic update let a caller pass them through,
// a row could be repointed so a client-B user becomes an active servicer of a
// client-A subscription. We re-read the persisted row and force both identity
// fields back to their persisted values before the write; only mutable fields
// (is_owner / active) actually change.
func (uc *UpdateSubscriptionWorkspaceUserUseCase) Execute(ctx context.Context, req *subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserRequest) (*subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionWorkspaceUser, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Preserve the identity fields (subscription_id, client_id, workspace_user_id)
	// from the persisted row — subscription_id/client_id are stamped only at create
	// from subscription.client_id, and workspace_user_id is fixed at create (the
	// on-account-team precheck only runs at create), so an update must not repoint it.
	if err := uc.preserveIdentity(ctx, req.Data); err != nil {
		return nil, err
	}

	uc.enrich(req.Data)

	resp, err := uc.repositories.SubscriptionWorkspaceUser.UpdateSubscriptionWorkspaceUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.update_failed", "Subscription workspace user update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// preserveIdentity re-reads the persisted row and forces subscription_id,
// client_id, AND workspace_user_id back to their persisted (create-time-stamped)
// values, so the generic update cannot repoint the junction's composite-FK
// identity across clients nor repoint the servicer to a different workspace_user
// within the same client. workspace_user_id is an identity field: changing the
// servicer must be delete+create (which re-runs the composite-FK + on-account-team
// precheck), never an update. is_owner and active remain mutable.
func (uc *UpdateSubscriptionWorkspaceUserUseCase) preserveIdentity(ctx context.Context, swu *subscriptionworkspaceuserpb.SubscriptionWorkspaceUser) error {
	readResp, err := uc.repositories.SubscriptionWorkspaceUser.ReadSubscriptionWorkspaceUser(ctx, &subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserRequest{
		Data: &subscriptionworkspaceuserpb.SubscriptionWorkspaceUser{Id: swu.Id},
	})
	if err != nil {
		return err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_workspace_user.errors.not_found", "Subscription workspace user not found [DEFAULT]"))
	}
	current := readResp.Data[0]
	swu.SubscriptionId = current.SubscriptionId
	swu.ClientId = current.ClientId
	swu.WorkspaceUserId = current.WorkspaceUserId
	return nil
}

func (uc *UpdateSubscriptionWorkspaceUserUseCase) validateInput(ctx context.Context, req *subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserRequest) error {
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

func (uc *UpdateSubscriptionWorkspaceUserUseCase) enrich(swu *subscriptionworkspaceuserpb.SubscriptionWorkspaceUser) {
	now := time.Now()
	swu.DateModified = &[]int64{now.UnixMilli()}[0]
	swu.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
