package subscription

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// trailingCodeRe matches the trailing " [CODE]" segment in a subscription
// name (e.g. "Advisory Monthly [MQJK48P]"). Used by the update use case to
// rewrite the bracketed code without rebuilding the plan-derived prefix.
var trailingCodeRe = regexp.MustCompile(`\s*\[[^\]]*\]\s*$`)

// rewriteNameWithCode strips any trailing "[…]" segment from name and appends
// " [newCode]". Empty newCode returns the base name with the bracket stripped.
// Empty base + non-empty newCode returns "[newCode]".
func rewriteNameWithCode(name, newCode string) string {
	base := strings.TrimSpace(trailingCodeRe.ReplaceAllString(name, ""))
	if newCode == "" {
		return base
	}
	if base == "" {
		return "[" + newCode + "]"
	}
	return base + " [" + newCode + "]"
}

type UpdateSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

type UpdateSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionUpdate); err != nil {
		return nil, err
	}

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
	// First, check if the subscription exists and capture the prior values
	// so we can detect a code change and rewrite the bracketed segment in
	// the name (e.g. "Advisory Monthly [OLD]" → "Advisory Monthly [NEW]")
	// without forcing the caller to rebuild the plan-derived prefix.
	readResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.not_found", "[ERR-DEFAULT] Subscription not found"))
	}

	var existing *subscriptionpb.Subscription
	if readResp != nil && len(readResp.GetData()) > 0 {
		existing = readResp.GetData()[0]
	}

	// If the code is being changed, rewrite the trailing "[…]" segment in
	// the name. We rewrite using whichever name is the "current" one — the
	// request's name when supplied, otherwise the existing record's name —
	// so partial updates that touch only the code still produce a coherent
	// final name. No-op when the code is unchanged.
	if existing != nil {
		newCode := enrichedSubscription.GetCode()
		oldCode := existing.GetCode()
		if newCode != oldCode {
			baseName := enrichedSubscription.GetName()
			if baseName == "" {
				baseName = existing.GetName()
			}
			rewritten := rewriteNameWithCode(baseName, newCode)
			enrichedSubscription.Name = rewritten
		}
	}

	resp, err := uc.repositories.Subscription.UpdateSubscription(ctx, &subscriptionpb.UpdateSubscriptionRequest{
		Data: enrichedSubscription,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.update_failed", "[ERR-DEFAULT] Subscription update failed"))
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_required", "[ERR-DEFAULT] Subscription ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
// Note: Only validates fields that are provided to support partial updates
// from workflow orchestration where only specific fields are updated
func (uc *UpdateSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Business rule: Required data validation
	if subscription == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", "[ERR-DEFAULT] Subscription data is required"))
	}

	// ID is always required for updates
	if subscription.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_required", "[ERR-DEFAULT] Subscription ID is required"))
	}
	if len(subscription.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_too_short", "[ERR-DEFAULT] Subscription ID is too short"))
	}

	// Validate Name only if provided
	if subscription.Name != "" {
		if len(subscription.Name) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_short", "[ERR-DEFAULT] Subscription name is too short"))
		}
		if len(subscription.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.name_too_long", "[ERR-DEFAULT] Subscription name is too long"))
		}
	}

	// Validate PricePlanId only if provided
	if subscription.PricePlanId != "" && len(subscription.PricePlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.price_plan_id_too_short", "[ERR-DEFAULT] Price plan ID is too short"))
	}

	// Validate ClientId only if provided
	if subscription.ClientId != "" && len(subscription.ClientId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.client_id_too_short", "[ERR-DEFAULT] Client ID is too short"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist.
//
// Plan §3.3 — when the chosen PricePlan is client-scoped (client_id != ""), it
// must match the subscription's client_id. The check resolves the
// subscription's effective client_id from the request body when present, else
// from the existing record (partial updates that change PricePlan but not
// ClientId still need to be gated).
func (uc *UpdateSubscriptionUseCase) validateEntityReferences(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Validate PricePlan entity reference
	if subscription.PricePlanId != "" {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: subscription.PricePlanId},
		})
		if err != nil || pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_found", "[ERR-DEFAULT] Price plan not found"))
		}
		if !pricePlan.Data[0].Active {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.price_plan_not_active", "[ERR-DEFAULT] Price plan is not active"))
		}

		// §3.3 — client-scope mismatch hard reject. Resolve effective
		// client_id (request body wins; fall back to existing record).
		ppClientID := pricePlan.Data[0].GetClientId()
		if ppClientID != "" {
			effectiveClientID := subscription.ClientId
			if effectiveClientID == "" && subscription.Id != "" {
				if existingResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{Id: subscription.Id},
				}); err == nil && existingResp != nil && len(existingResp.GetData()) > 0 {
					effectiveClientID = existingResp.GetData()[0].GetClientId()
				}
			}
			if ppClientID != effectiveClientID {
				return errors.New(contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.TranslationService,
					"subscription.errors.planClientMismatch",
					"This package belongs to a different client and cannot be attached here. [DEFAULT]",
				))
			}
		}
	}

	// Validate Client entity reference
	if subscription.ClientId != "" {
		client, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: subscription.ClientId},
		})
		if err != nil || client == nil || client.Data == nil || len(client.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_found", "[ERR-DEFAULT] Client not found"))
		}
		if !client.Data[0].Active {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.client_not_active", "[ERR-DEFAULT] Client is not active"))
		}
	}

	return nil
}
