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

type ListSupplierProductPlansRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type ListSupplierProductPlansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ListSupplierProductPlansUseCase struct {
	repositories ListSupplierProductPlansRepositories
	services     ListSupplierProductPlansServices
}

func NewListSupplierProductPlansUseCase(
	repositories ListSupplierProductPlansRepositories,
	services ListSupplierProductPlansServices,
) *ListSupplierProductPlansUseCase {
	return &ListSupplierProductPlansUseCase{repositories: repositories, services: services}
}

func (uc *ListSupplierProductPlansUseCase) Execute(ctx context.Context, req *supplierproductplanpb.ListSupplierProductPlansRequest) (*supplierproductplanpb.ListSupplierProductPlansResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductPlan, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.ListSupplierProductPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.errors.list_failed", "supplier product plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
