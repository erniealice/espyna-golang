package subscription_group_workspace_user

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

type UpdateSubscriptionGroupWorkspaceUserRepositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type UpdateSubscriptionGroupWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSubscriptionGroupWorkspaceUserUseCase struct {
	repositories UpdateSubscriptionGroupWorkspaceUserRepositories
	services     UpdateSubscriptionGroupWorkspaceUserServices
}

func NewUpdateSubscriptionGroupWorkspaceUserUseCase(r UpdateSubscriptionGroupWorkspaceUserRepositories, s UpdateSubscriptionGroupWorkspaceUserServices) *UpdateSubscriptionGroupWorkspaceUserUseCase {
	return &UpdateSubscriptionGroupWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupWorkspaceUserRequest) (*pb.UpdateSubscriptionGroupWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupWorkspaceUser, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.SubscriptionGroupWorkspaceUser.UpdateSubscriptionGroupWorkspaceUser(ctx, req)
}
