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

type CreateSubscriptionGroupWorkspaceUserRepositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type CreateSubscriptionGroupWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupWorkspaceUserUseCase struct {
	repositories CreateSubscriptionGroupWorkspaceUserRepositories
	services     CreateSubscriptionGroupWorkspaceUserServices
}

func NewCreateSubscriptionGroupWorkspaceUserUseCase(r CreateSubscriptionGroupWorkspaceUserRepositories, s CreateSubscriptionGroupWorkspaceUserServices) *CreateSubscriptionGroupWorkspaceUserUseCase {
	return &CreateSubscriptionGroupWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupWorkspaceUserRequest) (*pb.CreateSubscriptionGroupWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupWorkspaceUser, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.SubscriptionGroupWorkspaceUser.CreateSubscriptionGroupWorkspaceUser(ctx, req)
}

func (uc *CreateSubscriptionGroupWorkspaceUserUseCase) enrich(data *pb.SubscriptionGroupWorkspaceUser) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
