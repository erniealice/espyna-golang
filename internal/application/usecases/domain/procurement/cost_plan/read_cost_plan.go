package cost_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type ReadCostPlanRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type ReadCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadCostPlanUseCase struct {
	repositories ReadCostPlanRepositories
	services     ReadCostPlanServices
}

func NewReadCostPlanUseCase(
	repositories ReadCostPlanRepositories,
	services ReadCostPlanServices,
) *ReadCostPlanUseCase {
	return &ReadCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *ReadCostPlanUseCase) Execute(ctx context.Context, req *costplanpb.ReadCostPlanRequest) (*costplanpb.ReadCostPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CostPlan,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	return uc.repositories.CostPlan.ReadCostPlan(ctx, req)
}
