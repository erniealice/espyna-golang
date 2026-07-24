package subscription_group_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

type ListSubscriptionGroupProductPlansRepositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
}

type ListSubscriptionGroupProductPlansServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListSubscriptionGroupProductPlansUseCase struct {
	repositories ListSubscriptionGroupProductPlansRepositories
	services     ListSubscriptionGroupProductPlansServices
}

func NewListSubscriptionGroupProductPlansUseCase(r ListSubscriptionGroupProductPlansRepositories, s ListSubscriptionGroupProductPlansServices) *ListSubscriptionGroupProductPlansUseCase {
	return &ListSubscriptionGroupProductPlansUseCase{repositories: r, services: s}
}

func (uc *ListSubscriptionGroupProductPlansUseCase) Execute(ctx context.Context, req *pb.ListSubscriptionGroupProductPlansRequest) (*pb.ListSubscriptionGroupProductPlansResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupProductPlan.ListSubscriptionGroupProductPlans(ctx, req)
}
