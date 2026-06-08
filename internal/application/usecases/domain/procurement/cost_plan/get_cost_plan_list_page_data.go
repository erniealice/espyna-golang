package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type GetCostPlanListPageDataRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type GetCostPlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CostPlan, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.validation.request_required", "request is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *costplanpb.GetCostPlanListPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostPlan.GetCostPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "cost_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load cost plan list: %w"), err)
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
