package supplier_product_cost_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type UpdateSupplierProductCostPlanRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type UpdateSupplierProductCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type UpdateSupplierProductCostPlanUseCase struct {
	repositories UpdateSupplierProductCostPlanRepositories
	services     UpdateSupplierProductCostPlanServices
}

func NewUpdateSupplierProductCostPlanUseCase(
	repositories UpdateSupplierProductCostPlanRepositories,
	services UpdateSupplierProductCostPlanServices,
) *UpdateSupplierProductCostPlanUseCase {
	return &UpdateSupplierProductCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *UpdateSupplierProductCostPlanUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.UpdateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.UpdateSupplierProductCostPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductCostPlan, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.id_required", "supplier product cost plan ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierProductCostPlan.UpdateSupplierProductCostPlan(ctx, req)
}
