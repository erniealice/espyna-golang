package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type GetSupplierPlanListPageDataRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type GetSupplierPlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type GetSupplierPlanListPageDataUseCase struct {
	repositories GetSupplierPlanListPageDataRepositories
	services     GetSupplierPlanListPageDataServices
}

func NewGetSupplierPlanListPageDataUseCase(
	repositories GetSupplierPlanListPageDataRepositories,
	services GetSupplierPlanListPageDataServices,
) *GetSupplierPlanListPageDataUseCase {
	return &GetSupplierPlanListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierPlanListPageDataUseCase) Execute(ctx context.Context, req *supplierplanpb.GetSupplierPlanListPageDataRequest) (*supplierplanpb.GetSupplierPlanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySupplierPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.request_required", "request is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *supplierplanpb.GetSupplierPlanListPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierPlan.GetSupplierPlanListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "supplier_plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier plan list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierPlan.GetSupplierPlanListPageData(ctx, req)
}
