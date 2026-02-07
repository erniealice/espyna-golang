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

type CreateSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

type CreateSubscriptionServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
	IDService          ports.IDService
}

// CreateSubscriptionUseCase handles the business logic for creating subscriptions
type CreateSubscriptionUseCase struct {
	repositories CreateSubscriptionRepositories
	services     CreateSubscriptionServices
}

// NewCreateSubscriptionUseCase creates a new CreateSubscriptionUseCase
func NewCreateSubscriptionUseCase(
	repositories CreateSubscriptionRepositories,
	services CreateSubscriptionServices,
) *CreateSubscriptionUseCase {
	return &CreateSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create subscription operation
func (uc *CreateSubscriptionUseCase) Execute(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", ""))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
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

// executeWithTransaction executes subscription creation within a transaction
func (uc *CreateSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.CreateSubscriptionResponse, error) {
	var result *subscriptionpb.CreateSubscriptionResponse
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

// executeCore contains the core business logic for creating a subscription
func (uc *CreateSubscriptionUseCase) executeCore(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest, enrichedSubscription *subscriptionpb.Subscription) (*subscriptionpb.CreateSubscriptionResponse, error) {
	resp, err := uc.repositories.Subscription.CreateSubscription(ctx, &subscriptionpb.CreateSubscriptionRequest{
		Data: enrichedSubscription,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.creation_failed", ""))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched subscription
func (uc *CreateSubscriptionUseCase) applyBusinessLogic(subscription *subscriptionpb.Subscription) *subscriptionpb.Subscription {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if subscription.Id == "" {
		subscription.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new subscriptions
	subscription.Active = true

	// Business logic: Set creation audit fields
	subscription.DateCreated = &[]int64{now.UnixMilli()}[0]
	subscription.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	subscription.DateModified = &[]int64{now.UnixMilli()}[0]
	subscription.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return subscription
}

// validateBusinessRules enforces business constraints
func (uc *CreateSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Business rule: Required data validation
	if subscription == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", ""))
	}
	if subscription.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_required", ""))
	}
	if subscription.PricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_required", ""))
	}
	if subscription.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_required", ""))
	}

	// Business rule: Name length constraints
	if len(subscription.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_short", ""))
	}

	if len(subscription.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_long", ""))
	}

	// Business rule: PricePlan ID format validation
	if len(subscription.PricePlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_too_short", ""))
	}

	// Business rule: Client ID format validation
	if len(subscription.ClientId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_too_short", ""))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateSubscriptionUseCase) validateEntityReferences(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	if subscription == nil {
		return nil // Should be caught by validateBusinessRules
	}

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
