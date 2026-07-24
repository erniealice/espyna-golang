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

type ReadSubscriptionGroupProductPlanRepositories struct {
	SubscriptionGroupProductPlan pb.SubscriptionGroupProductPlanDomainServiceServer
}

type ReadSubscriptionGroupProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadSubscriptionGroupProductPlanUseCase struct {
	repositories ReadSubscriptionGroupProductPlanRepositories
	services     ReadSubscriptionGroupProductPlanServices
}

func NewReadSubscriptionGroupProductPlanUseCase(r ReadSubscriptionGroupProductPlanRepositories, s ReadSubscriptionGroupProductPlanServices) *ReadSubscriptionGroupProductPlanUseCase {
	return &ReadSubscriptionGroupProductPlanUseCase{repositories: r, services: s}
}

func (uc *ReadSubscriptionGroupProductPlanUseCase) Execute(ctx context.Context, req *pb.ReadSubscriptionGroupProductPlanRequest) (*pb.ReadSubscriptionGroupProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlan, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.SubscriptionGroupProductPlan.ReadSubscriptionGroupProductPlan(ctx, req)
}
