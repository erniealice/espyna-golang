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

type UpdateSubscriptionGroupMemberRepositories struct {
	SubscriptionGroupMember pb.SubscriptionGroupMemberDomainServiceServer
}

type UpdateSubscriptionGroupMemberServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSubscriptionGroupMemberUseCase struct {
	repositories UpdateSubscriptionGroupMemberRepositories
	services     UpdateSubscriptionGroupMemberServices
}

func NewUpdateSubscriptionGroupMemberUseCase(r UpdateSubscriptionGroupMemberRepositories, s UpdateSubscriptionGroupMemberServices) *UpdateSubscriptionGroupMemberUseCase {
	return &UpdateSubscriptionGroupMemberUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupMemberUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupMemberRequest) (*pb.UpdateSubscriptionGroupMemberResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupMember, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_member.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.SubscriptionGroupMember.UpdateSubscriptionGroupMember(ctx, req)
}
