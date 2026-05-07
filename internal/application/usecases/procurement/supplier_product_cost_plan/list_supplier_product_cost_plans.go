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

type ListSupplierProductCostPlansRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type ListSupplierProductCostPlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ListSupplierProductCostPlansUseCase struct {
	repositories ListSupplierProductCostPlansRepositories
	services     ListSupplierProductCostPlansServices
}

func NewListSupplierProductCostPlansUseCase(
	repositories ListSupplierProductCostPlansRepositories,
	services ListSupplierProductCostPlansServices,
) *ListSupplierProductCostPlansUseCase {
	return &ListSupplierProductCostPlansUseCase{repositories: repositories, services: services}
}

func (uc *ListSupplierProductCostPlansUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.ListSupplierProductCostPlansRequest) (*supplierproductcostplanpb.ListSupplierProductCostPlansResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_cost_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierProductCostPlan.ListSupplierProductCostPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_cost_plan.errors.list_failed", "supplier product cost plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
