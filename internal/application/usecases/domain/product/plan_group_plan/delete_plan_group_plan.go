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

type DeletePlanGroupPlanRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type DeletePlanGroupPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeletePlanGroupPlanUseCase struct {
	repositories DeletePlanGroupPlanRepositories
	services     DeletePlanGroupPlanServices
}

func NewDeletePlanGroupPlanUseCase(r DeletePlanGroupPlanRepositories, s DeletePlanGroupPlanServices) *DeletePlanGroupPlanUseCase {
	return &DeletePlanGroupPlanUseCase{repositories: r, services: s}
}

func (uc *DeletePlanGroupPlanUseCase) Execute(ctx context.Context, req *pb.DeletePlanGroupPlanRequest) (*pb.DeletePlanGroupPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroupPlan.DeletePlanGroupPlan(ctx, req)
}
