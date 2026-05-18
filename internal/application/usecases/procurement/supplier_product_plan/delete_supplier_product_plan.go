package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type DeleteSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type DeleteSupplierProductPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.DeleteSupplierProductPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.errors.deletion_failed", "supplier product plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
