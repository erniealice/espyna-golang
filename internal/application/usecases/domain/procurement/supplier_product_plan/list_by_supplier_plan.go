package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type ListBySupplierPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type ListBySupplierPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListBySupplierPlanUseCase struct {
	repositories ListBySupplierPlanRepositories
	services     ListBySupplierPlanServices
}

func NewListBySupplierPlanUseCase(
	repositories ListBySupplierPlanRepositories,
	services ListBySupplierPlanServices,
) *ListBySupplierPlanUseCase {
	return &ListBySupplierPlanUseCase{repositories: repositories, services: services}
}

func (uc *ListBySupplierPlanUseCase) Execute(ctx context.Context, req *supplierproductplanpb.ListSupplierProductPlansBySupplierPlanRequest) (*supplierproductplanpb.ListSupplierProductPlansBySupplierPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierProductPlan,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.request_required", "request is required"))
	}
	if req.SupplierPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.supplier_plan_id_required", "supplier plan ID is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.ListBySupplierPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.errors.list_by_supplier_plan_failed", "listing supplier product plans by supplier plan failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
