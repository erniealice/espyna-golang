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

type UpdateSubscriptionGroupRepositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer
}

type UpdateSubscriptionGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSubscriptionGroupUseCase struct {
	repositories UpdateSubscriptionGroupRepositories
	services     UpdateSubscriptionGroupServices
}

func NewUpdateSubscriptionGroupUseCase(r UpdateSubscriptionGroupRepositories, s UpdateSubscriptionGroupServices) *UpdateSubscriptionGroupUseCase {
	return &UpdateSubscriptionGroupUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupRequest) (*pb.UpdateSubscriptionGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroup, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.SubscriptionGroup.UpdateSubscriptionGroup(ctx, req)
}
