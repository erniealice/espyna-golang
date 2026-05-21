package cost_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type UpdateCostPlanRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type UpdateCostPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateCostPlanUseCase struct {
	repositories UpdateCostPlanRepositories
	services     UpdateCostPlanServices
}

func NewUpdateCostPlanUseCase(
	repositories UpdateCostPlanRepositories,
	services UpdateCostPlanServices,
) *UpdateCostPlanUseCase {
	return &UpdateCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *UpdateCostPlanUseCase) Execute(ctx context.Context, req *costplanpb.UpdateCostPlanRequest) (*costplanpb.UpdateCostPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.CostPlan.UpdateCostPlan(ctx, req)
}
