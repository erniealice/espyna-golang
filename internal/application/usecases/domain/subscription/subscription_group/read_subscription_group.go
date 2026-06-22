package subscription_group

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
)

type ReadSubscriptionGroupRepositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer
}

type ReadSubscriptionGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadSubscriptionGroupUseCase struct {
	repositories ReadSubscriptionGroupRepositories
	services     ReadSubscriptionGroupServices
}

func NewReadSubscriptionGroupUseCase(r ReadSubscriptionGroupRepositories, s ReadSubscriptionGroupServices) *ReadSubscriptionGroupUseCase {
	return &ReadSubscriptionGroupUseCase{repositories: r, services: s}
}

func (uc *ReadSubscriptionGroupUseCase) Execute(ctx context.Context, req *pb.ReadSubscriptionGroupRequest) (*pb.ReadSubscriptionGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroup, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroup.ReadSubscriptionGroup(ctx, req)
}
