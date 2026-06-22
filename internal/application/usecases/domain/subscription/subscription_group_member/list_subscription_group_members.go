package subscription_group_member

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
)

type ListSubscriptionGroupMembersRepositories struct {
	SubscriptionGroupMember pb.SubscriptionGroupMemberDomainServiceServer
}

type ListSubscriptionGroupMembersServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListSubscriptionGroupMembersUseCase struct {
	repositories ListSubscriptionGroupMembersRepositories
	services     ListSubscriptionGroupMembersServices
}

func NewListSubscriptionGroupMembersUseCase(r ListSubscriptionGroupMembersRepositories, s ListSubscriptionGroupMembersServices) *ListSubscriptionGroupMembersUseCase {
	return &ListSubscriptionGroupMembersUseCase{repositories: r, services: s}
}

func (uc *ListSubscriptionGroupMembersUseCase) Execute(ctx context.Context, req *pb.ListSubscriptionGroupMembersRequest) (*pb.ListSubscriptionGroupMembersResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupMember, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_member.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupMember.ListSubscriptionGroupMembers(ctx, req)
}
