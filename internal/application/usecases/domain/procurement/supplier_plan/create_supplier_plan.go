package supplier_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

type CreateSupplierPlanRepositories struct {
	SupplierPlan supplierplanpb.SupplierPlanDomainServiceServer
}

type CreateSupplierPlanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

type CreateSupplierPlanUseCase struct {
	repositories CreateSupplierPlanRepositories
	services     CreateSupplierPlanServices
}

func NewCreateSupplierPlanUseCase(
	repositories CreateSupplierPlanRepositories,
	services CreateSupplierPlanServices,
) *CreateSupplierPlanUseCase {
	return &CreateSupplierPlanUseCase{repositories: repositories, services: services}
}

func (uc *CreateSupplierPlanUseCase) Execute(ctx context.Context, req *supplierplanpb.CreateSupplierPlanRequest) (*supplierplanpb.CreateSupplierPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySupplierPlan, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.data_required", "supplier plan data is required"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.validation.name_required", "supplier plan name is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateSupplierPlanUseCase) executeWithTransaction(ctx context.Context, req *supplierplanpb.CreateSupplierPlanRequest) (*supplierplanpb.CreateSupplierPlanResponse, error) {
	var result *supplierplanpb.CreateSupplierPlanResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_plan.errors.creation_failed", "supplier plan creation failed")
			return fmt.Errorf("%s: %w", msg, err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateSupplierPlanUseCase) executeCore(ctx context.Context, req *supplierplanpb.CreateSupplierPlanRequest) (*supplierplanpb.CreateSupplierPlanResponse, error) {
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierPlan.CreateSupplierPlan(ctx, req)
}
