package supplier_product_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type UpdateSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type UpdateSupplierProductPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierProductPlan, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierProductPlan.UpdateSupplierProductPlan(ctx, req)
}
