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

type ReadPlanGroupPlanRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type ReadPlanGroupPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadPlanGroupPlanUseCase struct {
	repositories ReadPlanGroupPlanRepositories
	services     ReadPlanGroupPlanServices
}

func NewReadPlanGroupPlanUseCase(r ReadPlanGroupPlanRepositories, s ReadPlanGroupPlanServices) *ReadPlanGroupPlanUseCase {
	return &ReadPlanGroupPlanUseCase{repositories: r, services: s}
}

func (uc *ReadPlanGroupPlanUseCase) Execute(ctx context.Context, req *pb.ReadPlanGroupPlanRequest) (*pb.ReadPlanGroupPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroupPlan.ReadPlanGroupPlan(ctx, req)
}
