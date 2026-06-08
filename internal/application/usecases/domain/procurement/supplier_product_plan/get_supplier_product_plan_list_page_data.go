package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type GetSupplierProductPlanListPageDataRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type GetSupplierProductPlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductPlan, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.request_required", "request is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *supplierproductplanpb.GetSupplierProductPlanListPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierProductPlan.GetSupplierProductPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "supplier_product_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier product plan list: %w"), err)
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
