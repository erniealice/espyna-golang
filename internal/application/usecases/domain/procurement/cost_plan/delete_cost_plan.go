package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type DeleteCostPlanRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type DeleteCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteCostPlanUseCase struct {
	repositories DeleteCostPlanRepositories
	services     DeleteCostPlanServices
}

func NewDeleteCostPlanUseCase(
	repositories DeleteCostPlanRepositories,
	services DeleteCostPlanServices,
) *DeleteCostPlanUseCase {
	return &DeleteCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *DeleteCostPlanUseCase) Execute(ctx context.Context, req *costplanpb.DeleteCostPlanRequest) (*costplanpb.DeleteCostPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CostPlan,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	result, err := uc.repositories.CostPlan.DeleteCostPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.errors.deletion_failed", "cost plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
