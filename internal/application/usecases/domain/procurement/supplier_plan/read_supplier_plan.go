package supplier_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type ReadSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type ReadSupplierPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierPlan,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.id_required", "supplier plan ID is required"))
	}
	return uc.repositories.SupplierPlan.ReadSupplierPlan(ctx, req)
}
