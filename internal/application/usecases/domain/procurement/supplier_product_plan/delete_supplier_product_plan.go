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

type DeleteSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type DeleteSupplierProductPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type DeleteSupplierProductPlanUseCase struct {
	repositories DeleteSupplierProductPlanRepositories
	services     DeleteSupplierProductPlanServices
}

func NewDeleteSupplierProductPlanUseCase(
	repositories DeleteSupplierProductPlanRepositories,
	services DeleteSupplierProductPlanServices,
) *DeleteSupplierProductPlanUseCase {
	return &DeleteSupplierProductPlanUseCase{repositories: repositories, services: services}
}

func (uc *DeleteSupplierProductPlanUseCase) Execute(ctx context.Context, req *supplierproductplanpb.DeleteSupplierProductPlanRequest) (*supplierproductplanpb.DeleteSupplierProductPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductPlan, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.DeleteSupplierProductPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.errors.deletion_failed", "supplier product plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
