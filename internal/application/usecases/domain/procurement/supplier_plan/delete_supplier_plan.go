package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type DeleteSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type DeleteSupplierPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type DeleteSupplierPlanUseCase struct {
	repositories DeleteSupplierPlanRepositories
	services     DeleteSupplierPlanServices
}

func NewDeleteSupplierPlanUseCase(
	repositories DeleteSupplierPlanRepositories,
	services DeleteSupplierPlanServices,
) *DeleteSupplierPlanUseCase {
	return &DeleteSupplierPlanUseCase{repositories: repositories, services: services}
}

func (uc *DeleteSupplierPlanUseCase) Execute(ctx context.Context, req *supplierplanpb.DeleteSupplierPlanRequest) (*supplierplanpb.DeleteSupplierPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierPlan, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	result, err := uc.repositories.SupplierPlan.DeleteSupplierPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.errors.deletion_failed", "supplier plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
