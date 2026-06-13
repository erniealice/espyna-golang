package supplier_product_cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type DeleteSupplierProductCostPlanRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type DeleteSupplierProductCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteSupplierProductCostPlanUseCase struct {
	repositories DeleteSupplierProductCostPlanRepositories
	services     DeleteSupplierProductCostPlanServices
}

func NewDeleteSupplierProductCostPlanUseCase(
	repositories DeleteSupplierProductCostPlanRepositories,
	services DeleteSupplierProductCostPlanServices,
) *DeleteSupplierProductCostPlanUseCase {
	return &DeleteSupplierProductCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *DeleteSupplierProductCostPlanUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.DeleteSupplierProductCostPlanRequest) (*supplierproductcostplanpb.DeleteSupplierProductCostPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierProductCostPlan,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.id_required", "supplier product cost plan ID is required"))
	}
	result, err := uc.repositories.SupplierProductCostPlan.DeleteSupplierProductCostPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.errors.deletion_failed", "supplier product cost plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
