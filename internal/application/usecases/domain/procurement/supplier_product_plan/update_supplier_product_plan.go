package supplier_product_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type UpdateSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type UpdateSupplierProductPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateSupplierProductPlanUseCase struct {
	repositories UpdateSupplierProductPlanRepositories
	services     UpdateSupplierProductPlanServices
}

func NewUpdateSupplierProductPlanUseCase(
	repositories UpdateSupplierProductPlanRepositories,
	services UpdateSupplierProductPlanServices,
) *UpdateSupplierProductPlanUseCase {
	return &UpdateSupplierProductPlanUseCase{repositories: repositories, services: services}
}

func (uc *UpdateSupplierProductPlanUseCase) Execute(ctx context.Context, req *supplierproductplanpb.UpdateSupplierProductPlanRequest) (*supplierproductplanpb.UpdateSupplierProductPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierProductPlan.UpdateSupplierProductPlan(ctx, req)
}
