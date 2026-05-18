package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type DeleteCostPlanRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type DeleteCostPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	result, err := uc.repositories.CostPlan.DeleteCostPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.errors.deletion_failed", "cost plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
