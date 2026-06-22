package plan_group_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group_plan"
)

type ListPlanGroupPlansRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type ListPlanGroupPlansServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListPlanGroupPlansUseCase struct {
	repositories ListPlanGroupPlansRepositories
	services     ListPlanGroupPlansServices
}

func NewListPlanGroupPlansUseCase(r ListPlanGroupPlansRepositories, s ListPlanGroupPlansServices) *ListPlanGroupPlansUseCase {
	return &ListPlanGroupPlansUseCase{repositories: r, services: s}
}

func (uc *ListPlanGroupPlansUseCase) Execute(ctx context.Context, req *pb.ListPlanGroupPlansRequest) (*pb.ListPlanGroupPlansResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroupPlan.ListPlanGroupPlans(ctx, req)
}
