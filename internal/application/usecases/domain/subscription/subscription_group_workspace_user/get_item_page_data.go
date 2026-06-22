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

type GetSubscriptionGroupWorkspaceUserItemPageDataRepositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type GetSubscriptionGroupWorkspaceUserItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetSubscriptionGroupWorkspaceUserItemPageDataUseCase struct {
	repositories GetSubscriptionGroupWorkspaceUserItemPageDataRepositories
	services     GetSubscriptionGroupWorkspaceUserItemPageDataServices
}

func NewGetSubscriptionGroupWorkspaceUserItemPageDataUseCase(r GetSubscriptionGroupWorkspaceUserItemPageDataRepositories, s GetSubscriptionGroupWorkspaceUserItemPageDataServices) *GetSubscriptionGroupWorkspaceUserItemPageDataUseCase {
	return &GetSubscriptionGroupWorkspaceUserItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetSubscriptionGroupWorkspaceUserItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetSubscriptionGroupWorkspaceUserItemPageDataRequest) (*pb.GetSubscriptionGroupWorkspaceUserItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupWorkspaceUser, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupWorkspaceUser.GetSubscriptionGroupWorkspaceUserItemPageData(ctx, req)
}
