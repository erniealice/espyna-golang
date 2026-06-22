package subscription_group_member

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
)

type CreateSubscriptionGroupMemberRepositories struct {
	SubscriptionGroupMember pb.SubscriptionGroupMemberDomainServiceServer
}

type CreateSubscriptionGroupMemberServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupMemberUseCase struct {
	repositories CreateSubscriptionGroupMemberRepositories
	services     CreateSubscriptionGroupMemberServices
}

func NewCreateSubscriptionGroupMemberUseCase(r CreateSubscriptionGroupMemberRepositories, s CreateSubscriptionGroupMemberServices) *CreateSubscriptionGroupMemberUseCase {
	return &CreateSubscriptionGroupMemberUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupMemberUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupMemberRequest) (*pb.CreateSubscriptionGroupMemberResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupMember, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_member.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.SubscriptionGroupMember.CreateSubscriptionGroupMember(ctx, req)
}

func (uc *CreateSubscriptionGroupMemberUseCase) enrich(data *pb.SubscriptionGroupMember) {
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
