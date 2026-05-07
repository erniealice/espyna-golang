package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type ListSupplierPlansRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type ListSupplierPlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ListSupplierPlansUseCase struct {
	repositories ListSupplierPlansRepositories
	services     ListSupplierPlansServices
}

func NewListSupplierPlansUseCase(
	repositories ListSupplierPlansRepositories,
	services ListSupplierPlansServices,
) *ListSupplierPlansUseCase {
	return &ListSupplierPlansUseCase{repositories: repositories, services: services}
}

func (uc *ListSupplierPlansUseCase) Execute(ctx context.Context, req *supplierplanpb.ListSupplierPlansRequest) (*supplierplanpb.ListSupplierPlansResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierPlan.ListSupplierPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_plan.errors.list_failed", "supplier plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
