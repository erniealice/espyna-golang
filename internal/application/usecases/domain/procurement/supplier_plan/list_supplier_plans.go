package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type ListSupplierPlansRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type ListSupplierPlansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SupplierPlan, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierPlan.ListSupplierPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.errors.list_failed", "supplier plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
