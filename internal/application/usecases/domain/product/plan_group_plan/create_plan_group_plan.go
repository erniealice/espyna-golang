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

type CreatePlanGroupPlanRepositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type CreatePlanGroupPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreatePlanGroupPlanUseCase struct {
	repositories CreatePlanGroupPlanRepositories
	services     CreatePlanGroupPlanServices
}

func NewCreatePlanGroupPlanUseCase(r CreatePlanGroupPlanRepositories, s CreatePlanGroupPlanServices) *CreatePlanGroupPlanUseCase {
	return &CreatePlanGroupPlanUseCase{repositories: r, services: s}
}

func (uc *CreatePlanGroupPlanUseCase) Execute(ctx context.Context, req *pb.CreatePlanGroupPlanRequest) (*pb.CreatePlanGroupPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroupPlan, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group_plan.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.PlanGroupPlan.CreatePlanGroupPlan(ctx, req)
}

func (uc *CreatePlanGroupPlanUseCase) enrich(data *pb.PlanGroupPlan) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
