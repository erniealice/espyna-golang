package subscription

import (
	"context"
	"errors"
	"time"

	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
)

type UpdateSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

type UpdateSubscriptionServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateSubscriptionUseCase handles the business logic for updating subscriptions
type UpdateSubscriptionUseCase struct {
	repositories UpdateSubscriptionRepositories
	services     UpdateSubscriptionServices
}

// NewUpdateSubscriptionUseCase creates a new UpdateSubscriptionUseCase
func NewUpdateSubscriptionUseCase(
	repositories UpdateSubscriptionRepositories,
	services UpdateSubscriptionServices,
) *UpdateSubscriptionUseCase {
	return &UpdateSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update subscription operation
func (uc *UpdateSubscriptionUseCase) Execute(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedSubscription := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedSubscription)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedSubscription)
}

// executeWithTransaction executes subscription update within a transaction
func (uc *UpdateSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	var result *subscriptionpb.UpdateSubscriptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedSubscription)
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

// executeCore contains the core business logic for updating a subscription
func (uc *UpdateSubscriptionUseCase) executeCore(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	// First, check if the subscription exists
	_, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.not_found", ""))
	}

	resp, err := uc.repositories.Subscription.UpdateSubscription(ctx, &subscriptionpb.UpdateSubscriptionRequest{
		Data: enrichedSubscription,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.update_failed", ""))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched subscription
func (uc *UpdateSubscriptionUseCase) applyBusinessLogic(subscription *subscriptionpb.Subscription) *subscriptionpb.Subscription {
	now := time.Now()

	// Business logic: Update modification audit fields
	subscription.DateModified = &[]int64{now.UnixMilli()}[0]
	subscription.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return subscription
}

// validateInput validates the input request
func (uc *UpdateSubscriptionUseCase) validateInput(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints
// Note: Only validates fields that are provided to support partial updates
// from workflow orchestration where only specific fields are updated
func (uc *UpdateSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Business rule: Required data validation
	if subscription == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", ""))
	}

	// ID is always required for updates
	if subscription.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_required", ""))
	}
	if len(subscription.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_too_short", ""))
	}

	// Validate Name only if provided
	if subscription.Name != "" {
		if len(subscription.Name) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_short", ""))
		}
		if len(subscription.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_long", ""))
		}
	}

	// Validate PricePlanId only if provided
	if subscription.PricePlanId != "" && len(subscription.PricePlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_too_short", ""))
	}

	// Validate ClientId only if provided
	if subscription.ClientId != "" && len(subscription.ClientId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_too_short", ""))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateSubscriptionUseCase) validateEntityReferences(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Validate PricePlan entity reference
	if subscription.PricePlanId != "" {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: subscription.PricePlanId},
		})
		if err != nil || pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_found", ""))
		}
		if !pricePlan.Data[0].Active {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_active", ""))
		}
	}

	// Validate Client entity reference
	if subscription.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: subscription.ClientId},
		})
		if err != nil || client == nil || client.Data == nil || len(client.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_found", ""))
		}
		if !client.Data[0].Active {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_active", ""))
		}
	}

	return nil
}
