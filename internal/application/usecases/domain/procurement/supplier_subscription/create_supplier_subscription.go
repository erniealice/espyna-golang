package supplier_subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

type CreateSupplierSubscriptionRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	CostPlan             costplanpb.CostPlanDomainServiceServer   // Cross-domain: currency hard-block
	Workspace            workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block
}

type CreateSupplierSubscriptionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

type CreateSupplierSubscriptionUseCase struct {
	repositories CreateSupplierSubscriptionRepositories
	services     CreateSupplierSubscriptionServices
}

func NewCreateSupplierSubscriptionUseCase(
	repositories CreateSupplierSubscriptionRepositories,
	services CreateSupplierSubscriptionServices,
) *CreateSupplierSubscriptionUseCase {
	return &CreateSupplierSubscriptionUseCase{repositories: repositories, services: services}
}

func (uc *CreateSupplierSubscriptionUseCase) Execute(ctx context.Context, req *suppliersubscriptionpb.CreateSupplierSubscriptionRequest) (*suppliersubscriptionpb.CreateSupplierSubscriptionResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.SupplierSubscription,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.data_required", "supplier subscription data is required"))
	}
	if req.Data.CostPlanId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.cost_plan_id_required", "cost plan ID is required"))
	}
	if req.Data.SupplierId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.validation.supplier_id_required", "supplier ID is required"))
	}

	// Currency hard-block: the referenced CostPlan's billing_currency must match
	// the workspace's functional_currency to prevent cross-currency subscriptions.
	if uc.repositories.CostPlan != nil && uc.repositories.Workspace != nil {
		wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
		if wsID != "" {
			wsResp, wsErr := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
				Data: &workspacepb.Workspace{Id: wsID},
			})
			if wsErr == nil && wsResp != nil && len(wsResp.Data) > 0 {
				functionalCurrency := wsResp.Data[0].GetFunctionalCurrency()
				if functionalCurrency != "" {
					cpResp, cpErr := uc.repositories.CostPlan.ReadCostPlan(ctx, &costplanpb.ReadCostPlanRequest{
						Data: &costplanpb.CostPlan{Id: req.Data.CostPlanId},
					})
					if cpErr == nil && cpResp != nil && len(cpResp.Data) > 0 {
						if cpResp.Data[0].BillingCurrency != functionalCurrency {
							return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
								"supplier_subscription.errors.currency_mismatch",
								"cost plan billing currency must match workspace functional currency"))
						}
					}
				}
			}
		}
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateSupplierSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *suppliersubscriptionpb.CreateSupplierSubscriptionRequest) (*suppliersubscriptionpb.CreateSupplierSubscriptionResponse, error) {
	var result *suppliersubscriptionpb.CreateSupplierSubscriptionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_subscription.errors.creation_failed", "supplier subscription creation failed")
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

func (uc *CreateSupplierSubscriptionUseCase) executeCore(ctx context.Context, req *suppliersubscriptionpb.CreateSupplierSubscriptionRequest) (*suppliersubscriptionpb.CreateSupplierSubscriptionResponse, error) {
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.SupplierSubscription.CreateSupplierSubscription(ctx, req)
}
