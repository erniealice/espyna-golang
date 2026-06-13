package supplier_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type DeleteSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type DeleteSupplierPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierPlan,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	result, err := uc.repositories.SupplierPlan.DeleteSupplierPlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.errors.deletion_failed", "supplier plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
