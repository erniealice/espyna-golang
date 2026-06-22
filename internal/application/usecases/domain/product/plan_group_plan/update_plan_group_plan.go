package plan_group_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group_plan"
)

type UpdatePlanGroupPlanRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type UpdatePlanGroupPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdatePlanGroupPlanUseCase struct {
	repositories UpdatePlanGroupPlanRepositories
	services     UpdatePlanGroupPlanServices
}

func NewUpdatePlanGroupPlanUseCase(r UpdatePlanGroupPlanRepositories, s UpdatePlanGroupPlanServices) *UpdatePlanGroupPlanUseCase {
	return &UpdatePlanGroupPlanUseCase{repositories: r, services: s}
}

func (uc *UpdatePlanGroupPlanUseCase) Execute(ctx context.Context, req *pb.UpdatePlanGroupPlanRequest) (*pb.UpdatePlanGroupPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.PlanGroupPlan.UpdatePlanGroupPlan(ctx, req)
}
