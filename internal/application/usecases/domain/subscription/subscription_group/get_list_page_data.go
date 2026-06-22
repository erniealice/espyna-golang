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

type GetSubscriptionGroupListPageDataRepositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer
}

type GetSubscriptionGroupListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetSubscriptionGroupListPageDataUseCase struct {
	repositories GetSubscriptionGroupListPageDataRepositories
	services     GetSubscriptionGroupListPageDataServices
}

func NewGetSubscriptionGroupListPageDataUseCase(r GetSubscriptionGroupListPageDataRepositories, s GetSubscriptionGroupListPageDataServices) *GetSubscriptionGroupListPageDataUseCase {
	return &GetSubscriptionGroupListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetSubscriptionGroupListPageDataUseCase) Execute(ctx context.Context, req *pb.GetSubscriptionGroupListPageDataRequest) (*pb.GetSubscriptionGroupListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroup, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroup.GetSubscriptionGroupListPageData(ctx, req)
}
