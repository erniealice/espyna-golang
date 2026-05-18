package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type GetSupplierPlanItemPageDataRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type GetSupplierPlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetSupplierPlanItemPageDataUseCase struct {
	repositories GetSupplierPlanItemPageDataRepositories
	services     GetSupplierPlanItemPageDataServices
}

func NewGetSupplierPlanItemPageDataUseCase(
	repositories GetSupplierPlanItemPageDataRepositories,
	services GetSupplierPlanItemPageDataServices,
) *GetSupplierPlanItemPageDataUseCase {
	return &GetSupplierPlanItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetSupplierPlanItemPageDataUseCase) Execute(ctx context.Context, req *supplierplanpb.GetSupplierPlanItemPageDataRequest) (*supplierplanpb.GetSupplierPlanItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.SupplierPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *supplierplanpb.GetSupplierPlanItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierPlan.GetSupplierPlanItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier_plan.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier plan details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierPlan.GetSupplierPlanItemPageData(ctx, req)
}
