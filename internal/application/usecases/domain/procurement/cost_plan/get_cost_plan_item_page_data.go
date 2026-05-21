package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type GetCostPlanItemPageDataRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type GetCostPlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetCostPlanItemPageDataUseCase struct {
	repositories GetCostPlanItemPageDataRepositories
	services     GetCostPlanItemPageDataServices
}

func NewGetCostPlanItemPageDataUseCase(
	repositories GetCostPlanItemPageDataRepositories,
	services GetCostPlanItemPageDataServices,
) *GetCostPlanItemPageDataUseCase {
	return &GetCostPlanItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetCostPlanItemPageDataUseCase) Execute(ctx context.Context, req *costplanpb.GetCostPlanItemPageDataRequest) (*costplanpb.GetCostPlanItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.CostPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.id_required", "cost plan ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *costplanpb.GetCostPlanItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostPlan.GetCostPlanItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "cost_plan.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load cost plan details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.CostPlan.GetCostPlanItemPageData(ctx, req)
}
