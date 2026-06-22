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

type ListSubscriptionGroupWorkspaceUsersRepositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type ListSubscriptionGroupWorkspaceUsersServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListSubscriptionGroupWorkspaceUsersUseCase struct {
	repositories ListSubscriptionGroupWorkspaceUsersRepositories
	services     ListSubscriptionGroupWorkspaceUsersServices
}

func NewListSubscriptionGroupWorkspaceUsersUseCase(r ListSubscriptionGroupWorkspaceUsersRepositories, s ListSubscriptionGroupWorkspaceUsersServices) *ListSubscriptionGroupWorkspaceUsersUseCase {
	return &ListSubscriptionGroupWorkspaceUsersUseCase{repositories: r, services: s}
}

func (uc *ListSubscriptionGroupWorkspaceUsersUseCase) Execute(ctx context.Context, req *pb.ListSubscriptionGroupWorkspaceUsersRequest) (*pb.ListSubscriptionGroupWorkspaceUsersResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupWorkspaceUser, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupWorkspaceUser.ListSubscriptionGroupWorkspaceUsers(ctx, req)
}
