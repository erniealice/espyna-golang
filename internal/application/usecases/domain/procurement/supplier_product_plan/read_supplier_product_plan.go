package supplier_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type ReadSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type ReadSupplierProductPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ReadSupplierProductPlanUseCase struct {
	repositories ReadSupplierProductPlanRepositories
	services     ReadSupplierProductPlanServices
}

func NewReadSupplierProductPlanUseCase(
	repositories ReadSupplierProductPlanRepositories,
	services ReadSupplierProductPlanServices,
) *ReadSupplierProductPlanUseCase {
	return &ReadSupplierProductPlanUseCase{repositories: repositories, services: services}
}

func (uc *ReadSupplierProductPlanUseCase) Execute(ctx context.Context, req *supplierproductplanpb.ReadSupplierProductPlanRequest) (*supplierproductplanpb.ReadSupplierProductPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySupplierProductPlan, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_product_plan.validation.id_required", "supplier product plan ID is required"))
	}
	return uc.repositories.SupplierProductPlan.ReadSupplierProductPlan(ctx, req)
}
