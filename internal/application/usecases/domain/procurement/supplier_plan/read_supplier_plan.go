package supplier_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type ReadSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type ReadSupplierPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadSupplierPlanUseCase struct {
	repositories ReadSupplierPlanRepositories
	services     ReadSupplierPlanServices
}

func NewReadSupplierPlanUseCase(
	repositories ReadSupplierPlanRepositories,
	services ReadSupplierPlanServices,
) *ReadSupplierPlanUseCase {
	return &ReadSupplierPlanUseCase{repositories: repositories, services: services}
}

func (uc *ReadSupplierPlanUseCase) Execute(ctx context.Context, req *supplierplanpb.ReadSupplierPlanRequest) (*supplierplanpb.ReadSupplierPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierPlan, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	return uc.repositories.SupplierPlan.ReadSupplierPlan(ctx, req)
}
