package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ListSubscriptionsByPricePlanRepositories groups repository dependencies for
// the reverse-index "subscriptions for a given price plan" query.
type ListSubscriptionsByPricePlanRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

// ListSubscriptionsByPricePlanServices groups service dependencies.
type ListSubscriptionsByPricePlanServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListSubscriptionsByPricePlanUseCase resolves every subscription whose
// price_plan_id matches the request, hydrated with Client + PricePlan + Plan.
// Used by the centymo price-plan detail "Engagements" tab — see
// docs/plan/20260504-price-plan-engagements-tab/.
type ListSubscriptionsByPricePlanUseCase struct {
	repositories ListSubscriptionsByPricePlanRepositories
	services     ListSubscriptionsByPricePlanServices
}

func NewListSubscriptionsByPricePlanUseCase(
	repositories ListSubscriptionsByPricePlanRepositories,
	services ListSubscriptionsByPricePlanServices,
) *ListSubscriptionsByPricePlanUseCase {
	return &ListSubscriptionsByPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ListSubscriptionsByPricePlanUseCase) Execute(
	ctx context.Context,
	req *subscriptionpb.ListSubscriptionsByPricePlanRequest,
) (*subscriptionpb.ListSubscriptionsByPricePlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntitySubscription, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *subscriptionpb.ListSubscriptionsByPricePlanResponse
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

func (uc *ListSubscriptionsByPricePlanUseCase) executeCore(
	ctx context.Context,
	req *subscriptionpb.ListSubscriptionsByPricePlanRequest,
) (*subscriptionpb.ListSubscriptionsByPricePlanResponse, error) {
	resp, err := uc.repositories.Subscription.ListSubscriptionsByPricePlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.errors.list_by_price_plan_failed",
			"failed to list subscriptions for price plan: %w",
		), err)
	}
	return resp, nil
}

func (uc *ListSubscriptionsByPricePlanUseCase) validateInput(
	ctx context.Context,
	req *subscriptionpb.ListSubscriptionsByPricePlanRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.validation.request_required",
			"request is required",
		))
	}
	if req.PricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.validation.price_plan_id_required",
			"price_plan_id is required",
		))
	}
	return nil
}
