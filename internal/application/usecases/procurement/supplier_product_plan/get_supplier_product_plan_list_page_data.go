package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type GetSupplierProductPlanListPageDataRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type GetSupplierProductPlanListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierProductPlanListPageDataUseCase struct {
	repositories GetSupplierProductPlanListPageDataRepositories
	services     GetSupplierProductPlanListPageDataServices
}

func NewGetSupplierProductPlanListPageDataUseCase(
	repositories GetSupplierProductPlanListPageDataRepositories,
	services GetSupplierProductPlanListPageDataServices,
) *GetSupplierProductPlanListPageDataUseCase {
	return &GetSupplierProductPlanListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierProductPlanListPageDataUseCase) Execute(ctx context.Context, req *supplierproductplanpb.GetSupplierProductPlanListPageDataRequest) (*supplierproductplanpb.GetSupplierProductPlanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.request_required", "request is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *supplierproductplanpb.GetSupplierProductPlanListPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductPlan.GetSupplierProductPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_product_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product plan list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierProductPlan.GetSupplierProductPlanListPageData(ctx, req)
}
