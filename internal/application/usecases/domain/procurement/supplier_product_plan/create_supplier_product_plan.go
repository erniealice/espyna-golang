package supplier_product_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
)

type CreateSupplierProductPlanRepositories struct {
	SupplierProductPlan supplierproductplanpb.SupplierProductPlanDomainServiceServer
}

type CreateSupplierProductPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type CreateSupplierProductPlanUseCase struct {
	repositories CreateSupplierProductPlanRepositories
	services     CreateSupplierProductPlanServices
}

func NewCreateSupplierProductPlanUseCase(
	repositories CreateSupplierProductPlanRepositories,
	services CreateSupplierProductPlanServices,
) *CreateSupplierProductPlanUseCase {
	return &CreateSupplierProductPlanUseCase{repositories: repositories, services: services}
}

func (uc *CreateSupplierProductPlanUseCase) Execute(ctx context.Context, req *supplierproductplanpb.CreateSupplierProductPlanRequest) (*supplierproductplanpb.CreateSupplierProductPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySupplierProductPlan, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.data_required", "supplier product plan data is required"))
	}
	if req.Data.SupplierPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.validation.supplier_plan_id_required", "supplier plan ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateSupplierProductPlanUseCase) executeWithTransaction(ctx context.Context, req *supplierproductplanpb.CreateSupplierProductPlanRequest) (*supplierproductplanpb.CreateSupplierProductPlanResponse, error) {
	var result *supplierproductplanpb.CreateSupplierProductPlanResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_product_plan.errors.creation_failed", "supplier product plan creation failed")
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

func (uc *CreateSupplierProductPlanUseCase) executeCore(ctx context.Context, req *supplierproductplanpb.CreateSupplierProductPlanRequest) (*supplierproductplanpb.CreateSupplierProductPlanResponse, error) {
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierProductPlan.CreateSupplierProductPlan(ctx, req)
}
