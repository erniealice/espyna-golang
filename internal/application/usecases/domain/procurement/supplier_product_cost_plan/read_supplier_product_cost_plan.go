package supplier_product_cost_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type ReadSupplierProductCostPlanRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type ReadSupplierProductCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductCostPlan, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.id_required", "supplier product cost plan ID is required"))
	}
	return uc.repositories.SupplierProductCostPlan.ReadSupplierProductCostPlan(ctx, req)
}
