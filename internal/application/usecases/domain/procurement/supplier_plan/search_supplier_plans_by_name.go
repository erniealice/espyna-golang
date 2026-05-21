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

type SearchSupplierPlansByNameRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type SearchSupplierPlansByNameServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type SearchSupplierPlansByNameUseCase struct {
	repositories SearchSupplierPlansByNameRepositories
	services     SearchSupplierPlansByNameServices
}

func NewSearchSupplierPlansByNameUseCase(
	repositories SearchSupplierPlansByNameRepositories,
	services SearchSupplierPlansByNameServices,
) *SearchSupplierPlansByNameUseCase {
	return &SearchSupplierPlansByNameUseCase{repositories: repositories, services: services}
}

func (uc *SearchSupplierPlansByNameUseCase) Execute(ctx context.Context, req *supplierplanpb.SearchSupplierPlansByNameRequest) (*supplierplanpb.SearchSupplierPlansByNameResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySupplierPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.SupplierPlan.SearchSupplierPlansByName(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.errors.search_failed", "supplier plan search failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
