package supplier_product_cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type GetSupplierProductCostPlanListPageDataRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type GetSupplierProductCostPlanListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierProductCostPlanListPageDataUseCase struct {
	repositories GetSupplierProductCostPlanListPageDataRepositories
	services     GetSupplierProductCostPlanListPageDataServices
}

func NewGetSupplierProductCostPlanListPageDataUseCase(
	repositories GetSupplierProductCostPlanListPageDataRepositories,
	services GetSupplierProductCostPlanListPageDataServices,
) *GetSupplierProductCostPlanListPageDataUseCase {
	return &GetSupplierProductCostPlanListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierProductCostPlanListPageDataUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataRequest) (*supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_cost_plan.validation.request_required", "request is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductCostPlan.GetSupplierProductCostPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_product_cost_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product cost plan list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierProductCostPlan.GetSupplierProductCostPlanListPageData(ctx, req)
}
