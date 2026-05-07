package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type ListSupplierProductPlansRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type ListSupplierProductPlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.ListSupplierProductPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.errors.list_failed", "supplier product plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
