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

type GetPlanGroupPlanItemPageDataRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type GetPlanGroupPlanItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetPlanGroupPlanItemPageDataUseCase struct {
	repositories GetPlanGroupPlanItemPageDataRepositories
	services     GetPlanGroupPlanItemPageDataServices
}

func NewGetPlanGroupPlanItemPageDataUseCase(r GetPlanGroupPlanItemPageDataRepositories, s GetPlanGroupPlanItemPageDataServices) *GetPlanGroupPlanItemPageDataUseCase {
	return &GetPlanGroupPlanItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetPlanGroupPlanItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetPlanGroupPlanItemPageDataRequest) (*pb.GetPlanGroupPlanItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroupPlan.GetPlanGroupPlanItemPageData(ctx, req)
}
