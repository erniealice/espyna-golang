package supplier_product_cost_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type ReadSupplierProductCostPlanRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type ReadSupplierProductCostPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadSupplierProductCostPlanUseCase struct {
	repositories ReadSupplierProductCostPlanRepositories
	services     ReadSupplierProductCostPlanServices
}

func NewReadSupplierProductCostPlanUseCase(
	repositories ReadSupplierProductCostPlanRepositories,
	services ReadSupplierProductCostPlanServices,
) *ReadSupplierProductCostPlanUseCase {
	return &ReadSupplierProductCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *ReadSupplierProductCostPlanUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.ReadSupplierProductCostPlanRequest) (*supplierproductcostplanpb.ReadSupplierProductCostPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductCostPlan, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_cost_plan.validation.id_required", "supplier product cost plan ID is required"))
	}
	return uc.repositories.SupplierProductCostPlan.ReadSupplierProductCostPlan(ctx, req)
}
