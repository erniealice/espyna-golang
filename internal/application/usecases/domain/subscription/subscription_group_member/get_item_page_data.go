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

type GetSubscriptionGroupMemberItemPageDataRepositories struct {
	SubscriptionGroupMember pb.SubscriptionGroupMemberDomainServiceServer
}

type GetSubscriptionGroupMemberItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetSubscriptionGroupMemberItemPageDataUseCase struct {
	repositories GetSubscriptionGroupMemberItemPageDataRepositories
	services     GetSubscriptionGroupMemberItemPageDataServices
}

func NewGetSubscriptionGroupMemberItemPageDataUseCase(r GetSubscriptionGroupMemberItemPageDataRepositories, s GetSubscriptionGroupMemberItemPageDataServices) *GetSubscriptionGroupMemberItemPageDataUseCase {
	return &GetSubscriptionGroupMemberItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetSubscriptionGroupMemberItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetSubscriptionGroupMemberItemPageDataRequest) (*pb.GetSubscriptionGroupMemberItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupMember, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_member.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupMember.GetSubscriptionGroupMemberItemPageData(ctx, req)
}
