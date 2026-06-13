package supplier_product_cost_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type CreateSupplierProductCostPlanRepositories struct {
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
}

type CreateSupplierProductCostPlanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

type CreateSupplierProductCostPlanUseCase struct {
	repositories CreateSupplierProductCostPlanRepositories
	services     CreateSupplierProductCostPlanServices
}

func NewCreateSupplierProductCostPlanUseCase(
	repositories CreateSupplierProductCostPlanRepositories,
	services CreateSupplierProductCostPlanServices,
) *CreateSupplierProductCostPlanUseCase {
	return &CreateSupplierProductCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *CreateSupplierProductCostPlanUseCase) Execute(ctx context.Context, req *supplierproductcostplanpb.CreateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.CreateSupplierProductCostPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierProductCostPlan,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.data_required", "supplier product cost plan data is required"))
	}
	if req.Data.SupplierProductPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.supplier_product_plan_id_required", "supplier product plan ID is required"))
	}
	if req.Data.CostPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.validation.cost_plan_id_required", "cost plan ID is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateSupplierProductCostPlanUseCase) executeWithTransaction(ctx context.Context, req *supplierproductcostplanpb.CreateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.CreateSupplierProductCostPlanResponse, error) {
	var result *supplierproductcostplanpb.CreateSupplierProductCostPlanResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_cost_plan.errors.creation_failed", "supplier product cost plan creation failed")
			return fmt.Errorf("%s: %w", msg, err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateSupplierProductCostPlanUseCase) executeCore(ctx context.Context, req *supplierproductcostplanpb.CreateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.CreateSupplierProductCostPlanResponse, error) {
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierProductCostPlan.CreateSupplierProductCostPlan(ctx, req)
}
