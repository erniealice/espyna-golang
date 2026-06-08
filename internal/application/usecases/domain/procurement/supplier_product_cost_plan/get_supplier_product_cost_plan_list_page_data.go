package supplier_product_cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type GetSupplierProductCostPlanListPageDataRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type GetSupplierProductCostPlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductCostPlan, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.request_required", "request is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductCostPlan.GetSupplierProductCostPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "supplier_product_cost_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product cost plan list: %w"), err)
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
