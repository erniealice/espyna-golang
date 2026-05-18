package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type GetSupplierProductPlanItemPageDataRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type GetSupplierProductPlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierProductPlanItemPageDataUseCase struct {
	repositories GetSupplierProductPlanItemPageDataRepositories
	services     GetSupplierProductPlanItemPageDataServices
}

func NewGetSupplierProductPlanItemPageDataUseCase(
	repositories GetSupplierProductPlanItemPageDataRepositories,
	services GetSupplierProductPlanItemPageDataServices,
) *GetSupplierProductPlanItemPageDataUseCase {
	return &GetSupplierProductPlanItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierProductPlanItemPageDataUseCase) Execute(ctx context.Context, req *supplierproductplanpb.GetSupplierProductPlanItemPageDataRequest) (*supplierproductplanpb.GetSupplierProductPlanItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.SupplierProductPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *supplierproductplanpb.GetSupplierProductPlanItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductPlan.GetSupplierProductPlanItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_product_plan.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product plan details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierProductPlan.GetSupplierProductPlanItemPageData(ctx, req)
}
