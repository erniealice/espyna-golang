package supplier_product_cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type GetSupplierProductCostPlanItemPageDataRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type GetSupplierProductCostPlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierProductCostPlanItemPageDataUseCase struct {
	repositories GetSupplierProductCostPlanItemPageDataRepositories
	services     GetSupplierProductCostPlanItemPageDataServices
}

func NewGetSupplierProductCostPlanItemPageDataUseCase(
	repositories GetSupplierProductCostPlanItemPageDataRepositories,
	services GetSupplierProductCostPlanItemPageDataServices,
) *GetSupplierProductCostPlanItemPageDataUseCase {
	return &GetSupplierProductCostPlanItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierProductCostPlanItemPageDataUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataRequest) (*supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.SupplierProductCostPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_cost_plan.validation.id_required", "supplier product cost plan ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductCostPlan.GetSupplierProductCostPlanItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_product_cost_plan.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product cost plan details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierProductCostPlan.GetSupplierProductCostPlanItemPageData(ctx, req)
}
