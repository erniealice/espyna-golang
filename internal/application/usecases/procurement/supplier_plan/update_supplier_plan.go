package supplier_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type UpdateSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type UpdateSupplierPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateSupplierPlanUseCase struct {
	repositories UpdateSupplierPlanRepositories
	services     UpdateSupplierPlanServices
}

func NewUpdateSupplierPlanUseCase(
	repositories UpdateSupplierPlanRepositories,
	services UpdateSupplierPlanServices,
) *UpdateSupplierPlanUseCase {
	return &UpdateSupplierPlanUseCase{repositories: repositories, services: services}
}

func (uc *UpdateSupplierPlanUseCase) Execute(ctx context.Context, req *supplierplanpb.UpdateSupplierPlanRequest) (*supplierplanpb.UpdateSupplierPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierPlan, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.name_required", "supplier plan name is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierPlan.UpdateSupplierPlan(ctx, req)
}
