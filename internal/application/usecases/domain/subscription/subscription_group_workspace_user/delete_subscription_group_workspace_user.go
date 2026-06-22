package subscription_group_workspace_user

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

type DeleteSubscriptionGroupWorkspaceUserRepositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type DeleteSubscriptionGroupWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteSubscriptionGroupWorkspaceUserUseCase struct {
	repositories DeleteSubscriptionGroupWorkspaceUserRepositories
	services     DeleteSubscriptionGroupWorkspaceUserServices
}

func NewDeleteSubscriptionGroupWorkspaceUserUseCase(r DeleteSubscriptionGroupWorkspaceUserRepositories, s DeleteSubscriptionGroupWorkspaceUserServices) *DeleteSubscriptionGroupWorkspaceUserUseCase {
	return &DeleteSubscriptionGroupWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *DeleteSubscriptionGroupWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.DeleteSubscriptionGroupWorkspaceUserRequest) (*pb.DeleteSubscriptionGroupWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupWorkspaceUser, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupWorkspaceUser.DeleteSubscriptionGroupWorkspaceUser(ctx, req)
}
