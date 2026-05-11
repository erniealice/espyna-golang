package cost_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type CreateCostPlanRepositories struct {
	CostPlan  costplanpb.CostPlanDomainServiceServer
	Workspace workspacepb.WorkspaceDomainServiceServer // Cross-domain: for currency hard-block
}

type CreateCostPlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type CreateCostPlanUseCase struct {
	repositories CreateCostPlanRepositories
	services     CreateCostPlanServices
}

func NewCreateCostPlanUseCase(
	repositories CreateCostPlanRepositories,
	services CreateCostPlanServices,
) *CreateCostPlanUseCase {
	return &CreateCostPlanUseCase{repositories: repositories, services: services}
}

func (uc *CreateCostPlanUseCase) Execute(ctx context.Context, req *costplanpb.CreateCostPlanRequest) (*costplanpb.CreateCostPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.data_required", "cost plan data is required"))
	}
	if req.Data.SupplierPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.supplier_plan_id_required", "supplier plan ID is required"))
	}
	if req.Data.BillingCurrency == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.billing_currency_required", "billing currency is required"))
	}

	// Currency hard-block: billing_currency must match workspace functional_currency
	if uc.repositories.Workspace != nil {
		wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
		if wsID != "" {
			wsResp, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
				Data: &workspacepb.Workspace{Id: wsID},
			})
			if err == nil && wsResp != nil && len(wsResp.Data) > 0 {
				functionalCurrency := wsResp.Data[0].GetFunctionalCurrency()
				if functionalCurrency != "" && req.Data.BillingCurrency != functionalCurrency {
					return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
						"cost_plan.errors.currency_mismatch",
						"billing currency must match workspace functional currency"))
				}
			}
		}
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateCostPlanUseCase) executeWithTransaction(ctx context.Context, req *costplanpb.CreateCostPlanRequest) (*costplanpb.CreateCostPlanResponse, error) {
	var result *costplanpb.CreateCostPlanResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.errors.creation_failed", "cost plan creation failed")
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

func (uc *CreateCostPlanUseCase) executeCore(ctx context.Context, req *costplanpb.CreateCostPlanRequest) (*costplanpb.CreateCostPlanResponse, error) {
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.CostPlan.CreateCostPlan(ctx, req)
}
