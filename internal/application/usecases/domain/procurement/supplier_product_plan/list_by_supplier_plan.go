package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type ListBySupplierPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type ListBySupplierPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.request_required", "request is required"))
	}
	if req.SupplierPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.supplier_plan_id_required", "supplier plan ID is required"))
	}
	result, err := uc.repositories.SupplierProductPlan.ListBySupplierPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.errors.list_by_supplier_plan_failed", "listing supplier product plans by supplier plan failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
