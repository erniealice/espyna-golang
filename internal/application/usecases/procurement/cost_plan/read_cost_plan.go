package cost_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type ReadCostPlanRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type ReadCostPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	return uc.repositories.CostPlan.ReadCostPlan(ctx, req)
}
