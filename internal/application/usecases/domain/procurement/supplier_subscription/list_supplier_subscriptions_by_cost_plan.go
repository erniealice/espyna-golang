package supplier_subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// ListSupplierSubscriptionsByCostPlanRepositories groups repository dependencies for
// the reverse-index "supplier subscriptions for a given cost plan" query.
type ListSupplierSubscriptionsByCostPlanRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

// ListSupplierSubscriptionsByCostPlanServices groups service dependencies.
type ListSupplierSubscriptionsByCostPlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListSupplierSubscriptionsByCostPlanUseCase resolves every supplier subscription whose
// cost_plan_id matches the request, hydrated with Supplier + CostPlan.
// Used by the cost-plan detail "Engagements" tab.
type ListSupplierSubscriptionsByCostPlanUseCase struct {
	repositories ListSupplierSubscriptionsByCostPlanRepositories
	services     ListSupplierSubscriptionsByCostPlanServices
}

func NewListSupplierSubscriptionsByCostPlanUseCase(
	repositories ListSupplierSubscriptionsByCostPlanRepositories,
	services ListSupplierSubscriptionsByCostPlanServices,
) *ListSupplierSubscriptionsByCostPlanUseCase {
	return &ListSupplierSubscriptionsByCostPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ListSupplierSubscriptionsByCostPlanUseCase) Execute(
	ctx context.Context,
	req *suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanRequest,
) (*suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySupplierSubscription, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *ListSupplierSubscriptionsByCostPlanUseCase) executeCore(
	ctx context.Context,
	req *suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanRequest,
) (*suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanResponse, error) {
	resp, err := uc.repositories.SupplierSubscription.ListSupplierSubscriptionsByCostPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"supplier_subscription.errors.list_by_cost_plan_failed",
			"failed to list supplier subscriptions for cost plan: %w",
		), err)
	}
	return resp, nil
}

func (uc *ListSupplierSubscriptionsByCostPlanUseCase) validateInput(
	ctx context.Context,
	req *suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"supplier_subscription.validation.request_required",
			"request is required",
		))
	}
	if req.CostPlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"supplier_subscription.validation.cost_plan_id_required",
			"cost_plan_id is required",
		))
	}
	return nil
}
