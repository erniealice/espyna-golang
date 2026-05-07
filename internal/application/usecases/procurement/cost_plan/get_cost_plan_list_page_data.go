package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type GetCostPlanListPageDataRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type GetCostPlanListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetCostPlanListPageDataUseCase struct {
	repositories GetCostPlanListPageDataRepositories
	services     GetCostPlanListPageDataServices
}

func NewGetCostPlanListPageDataUseCase(
	repositories GetCostPlanListPageDataRepositories,
	services GetCostPlanListPageDataServices,
) *GetCostPlanListPageDataUseCase {
	return &GetCostPlanListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetCostPlanListPageDataUseCase) Execute(ctx context.Context, req *costplanpb.GetCostPlanListPageDataRequest) (*costplanpb.GetCostPlanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.request_required", "request is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *costplanpb.GetCostPlanListPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostPlan.GetCostPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "cost_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load cost plan list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.CostPlan.GetCostPlanListPageData(ctx, req)
}
