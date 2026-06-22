package subscription_group

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
)

type CreateSubscriptionGroupRepositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer
}

type CreateSubscriptionGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupUseCase struct {
	repositories CreateSubscriptionGroupRepositories
	services     CreateSubscriptionGroupServices
}

func NewCreateSubscriptionGroupUseCase(r CreateSubscriptionGroupRepositories, s CreateSubscriptionGroupServices) *CreateSubscriptionGroupUseCase {
	return &CreateSubscriptionGroupUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupRequest) (*pb.CreateSubscriptionGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroup, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.SubscriptionGroup.CreateSubscriptionGroup(ctx, req)
}

func (uc *CreateSubscriptionGroupUseCase) enrich(data *pb.SubscriptionGroup) {
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
