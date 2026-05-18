package subscription

import (
	"context"
	"errors"
	"log"
	"time"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
)

type CreateSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

type CreateSubscriptionServices struct {
	AuthorizationService    ports.AuthorizationService
	TransactionService      ports.TransactionService
	TranslationService      ports.TranslationService
	IDService               ports.IDService
	JobTemplateInstantiator JobTemplateInstantiator
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
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation — also returns the PricePlan so we can read plan_id later.
	pricePlan, err := uc.validateEntityReferences(ctx, req.Data)
	if err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedSubscription := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	var resp *subscriptionpb.CreateSubscriptionResponse
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		resp, err = uc.executeWithTransaction(ctx, req, enrichedSubscription)
	} else {
		// Fallback to non-transactional execution
		resp, err = uc.executeCore(ctx, req, enrichedSubscription)
	}
	if err != nil {
		return nil, err
	}

	// After successful creation, instantiate jobs from the plan (best-effort, non-blocking).
	//
	// 2026-04-29 auto-spawn-jobs-from-subscription plan §5.1 — the operator's
	// "Spawn Jobs on Create" toggle is propagated from the centymo view layer
	// via context (see espyna shared/context/spawn_jobs.go). When unset, fall
	// back to true to preserve the legacy default-on behavior for any callers
	// that did not adopt the toggle yet.
	if uc.services.JobTemplateInstantiator != nil && pricePlan != nil {
		wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
		spawnJobs := true
		if override, set := contextutil.ExtractSpawnJobsOverride(ctx); set {
			spawnJobs = override
		}
		if jiErr := uc.services.JobTemplateInstantiator.InstantiateJobsFromPlan(
			ctx, pricePlan.PlanId, enrichedSubscription.ClientId, enrichedSubscription.Id, wsID, spawnJobs,
		); jiErr != nil {
			log.Printf("Warning: job instantiation failed for subscription %s: %v", enrichedSubscription.Id, jiErr)
			// Do not fail subscription creation — log and continue.
		}
	}

	return resp, nil
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
		log.Printf("CreateSubscription DB error: %v", err)
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.creation_failed", "[ERR-DEFAULT] Subscription creation failed"))
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}
	if subscription.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_required", "[ERR-DEFAULT] Subscription name is required"))
	}
	if subscription.PricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_required", "[ERR-DEFAULT] Price plan ID is required"))
	}
	if subscription.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_required", "[ERR-DEFAULT] Client ID is required"))
	}

	// Business rule: Name length constraints
	if len(subscription.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_short", "[ERR-DEFAULT] Subscription name is too short"))
	}

	if len(subscription.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_long", "[ERR-DEFAULT] Subscription name is too long"))
	}

	// Business rule: PricePlan ID format validation
	if len(subscription.PricePlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_too_short", "[ERR-DEFAULT] Price plan ID is too short"))
	}

	// Business rule: Client ID format validation
	if len(subscription.ClientId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_too_short", "[ERR-DEFAULT] Client ID is too short"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist.
// It returns the resolved PricePlan so the caller can access plan_id after validation.
//
// Plan §3.3 — when the chosen PricePlan is client-scoped (client_id != ""), it
// must match the subscription's client_id. Master PricePlans (client_id == "")
// remain attachable for any client.
func (uc *CreateSubscriptionUseCase) validateEntityReferences(ctx context.Context, subscription *subscriptionpb.Subscription) (*priceplanpb.PricePlan, error) {
	if subscription == nil {
		return nil, nil // Should be caught by validateBusinessRules
	}

	var resolvedPricePlan *priceplanpb.PricePlan

	// Validate PricePlan entity reference
	if subscription.PricePlanId != "" {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: subscription.PricePlanId},
		})
		if err != nil || pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_found", "[ERR-DEFAULT] Price plan not found"))
		}
		if !pricePlan.Data[0].Active {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_active", "[ERR-DEFAULT] Price plan is not active"))
		}
		resolvedPricePlan = pricePlan.Data[0]

		// §3.3 — client-scope mismatch hard reject.
		if ppClientID := resolvedPricePlan.GetClientId(); ppClientID != "" && ppClientID != subscription.ClientId {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.TranslationService,
				"subscription.errors.planClientMismatch",
				"This package belongs to a different client and cannot be attached here. [DEFAULT]",
			))
		}
	}

	// Validate Client entity reference
	if subscription.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: subscription.ClientId},
		})
		if err != nil || client == nil || client.Data == nil || len(client.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_found", "[ERR-DEFAULT] Client not found"))
		}
		if !client.Data[0].Active {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_active", "[ERR-DEFAULT] Client is not active"))
		}
	}

	return resolvedPricePlan, nil
}
