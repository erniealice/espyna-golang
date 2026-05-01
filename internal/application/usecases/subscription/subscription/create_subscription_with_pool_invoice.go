package subscription

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// CreateSubscriptionWithPoolInvoiceUseCase atomically creates a subscription
// AND its prepaid pool invoice in a single DB transaction (codex CRIT-5
// fix from the AD_HOC plan red-team review).
//
// Background. v1 ships a manual Pool-Generate-Invoice CTA on the Operations
// tab (Task 14). Operators have to click Generate after creating an AD_HOC ×
// TOTAL_PACKAGE subscription. That avoids the atomicity-at-signup risk codex
// flagged ("subscription can exist without the promised pool invoice")
// because the invoice is a separate explicit action — there is no
// "half-created pool subscription" to recover from.
//
// v1.5 (this use case) makes the create+invoice pair automatic for shops
// that prefer signup-time invoicing:
//
//	BEGIN
//	  InsertSubscription          ← uc.subscription.executeCore
//	  ExecuteAdHocPoolRecognize   ← uc.recognize.executeAdHocPool
//	    → Revenue + RevenueLineItem; idempotent on the
//	      "ad_hoc_pool_initial" notes marker
//	  if any error: ROLLBACK and return; the subscription does not exist
//	COMMIT
//	post-commit: best-effort job instantiation (unchanged from
//	             plain CreateSubscription)
//
// Shape:
//
//   - Non-AD_HOC × TOTAL_PACKAGE bookings delegate to the plain
//     CreateSubscriptionUseCase. This keeps the dispatch decision local to
//     this wrapper — service-admin can wire ALL bookings through this use
//     case and let it route internally.
//
//   - currency mismatch / GL write failure / use-case validation errors all
//     surface as transaction failures. Operators see the recognize error
//     (the wrapping use case returns it verbatim) and have to fix the
//     underlying issue before the customer is signed up.
//
// Wiring (v1.5.5 — not yet done; this commit ships the use case only):
//   - container.go must build a CreateSubscriptionWithPoolInvoiceUseCase
//     and pass it to the centymo subscription action handler instead of /
//     alongside the plain CreateSubscription
//   - the action handler dispatches based on the PricePlan it just resolved
//     for the request (already done in validateEntityReferences)
//
// See docs/plan/20260501-ad-hoc-subscription-billing/plan.md §4.1 + §19
// CRIT-5 for the full atomicity contract.
type CreateSubscriptionWithPoolInvoiceUseCase struct {
	subscription      *CreateSubscriptionUseCase
	recognizePoolFunc RecognizePoolInvoker
	services          CreateSubscriptionServices
}

// RecognizePoolInvoker is the narrow interface the wrapping use case needs
// from the recognize-revenue layer. The full RecognizeRevenueFromSubscription
// use case lives in revenue/revenue/, and this package cannot import it
// (circular dep), so callers (container) build a thin adapter that satisfies
// this contract by closing over the concrete use case.
//
// The adapter is responsible for:
//
//   - reading the price plan from the just-created subscription
//   - confirming kind == AD_HOC × basis == TOTAL_PACKAGE
//   - dispatching to the recognize use case's executeAdHocPool path
//
// On success the adapter returns nil. On any error, the wrapping use case
// rolls back the wrapping transaction.
type RecognizePoolInvoker interface {
	RecognizeAdHocPool(ctx context.Context, subscriptionID string) error
}

// NewCreateSubscriptionWithPoolInvoiceUseCase wires the wrapping use case.
//
// `recognize` may be nil — in that case the wrapping use case behaves like
// plain CreateSubscription (no pool recognize attempted). Operators see the
// same v1 manual-trigger contract.
func NewCreateSubscriptionWithPoolInvoiceUseCase(
	subscription *CreateSubscriptionUseCase,
	recognize RecognizePoolInvoker,
	services CreateSubscriptionServices,
) *CreateSubscriptionWithPoolInvoiceUseCase {
	return &CreateSubscriptionWithPoolInvoiceUseCase{
		subscription:      subscription,
		recognizePoolFunc: recognize,
		services:          services,
	}
}

// Execute performs the create+invoice round-trip atomically.
//
// Dispatch:
//
//	plain CreateSubscription path                    when the bound PricePlan
//	                                                 is not AD_HOC × TOTAL_PACKAGE,
//	                                                 OR when no recognize adapter
//	                                                 is wired (graceful fallback)
//
//	atomic create + recognize-pool path              when AD_HOC × TOTAL_PACKAGE
//	                                                 AND recognize adapter wired
//
// Returns the subscription create response. Subscription Id, audit timestamps,
// active flag, etc. are enriched identically to the plain path.
func (uc *CreateSubscriptionWithPoolInvoiceUseCase) Execute(
	ctx context.Context,
	req *subscriptionpb.CreateSubscriptionRequest,
) (*subscriptionpb.CreateSubscriptionResponse, error) {
	if uc.subscription == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.create_with_pool_invoice_unwired",
			"CreateSubscriptionWithPoolInvoice is not configured [DEFAULT]",
		))
	}

	// Authorization mirrors the plain path; centralised so the wrapping use
	// case is self-contained.
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.validation.data_required",
			"[ERR-DEFAULT] Subscription data is required",
		))
	}

	// Validate + enrich (mirrors CreateSubscription.Execute up to the tx call).
	if err := uc.subscription.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}
	pricePlan, err := uc.subscription.validateEntityReferences(ctx, req.Data)
	if err != nil {
		return nil, err
	}
	enriched := uc.subscription.applyBusinessLogic(req.Data)

	// Dispatch decision: only AD_HOC × TOTAL_PACKAGE PricePlans engage the
	// atomic-with-recognize path. Everything else delegates to the existing
	// plain path verbatim.
	useAtomicPool := isAdHocPool(pricePlan) && uc.recognizePoolFunc != nil

	var resp *subscriptionpb.CreateSubscriptionResponse
	if !useAtomicPool {
		// Plain path — delegate to CreateSubscription (which handles its own
		// transaction internally). Job instantiation runs inside Execute too,
		// so the caller doesn't need to repeat the post-commit logic.
		return uc.subscription.Execute(ctx, req)
	}

	// AD_HOC × TOTAL_PACKAGE atomic path: BOTH writes inside one transaction.
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.transaction_required_for_pool_invoice",
			"A transactional service is required for atomic pool-invoice subscription create [DEFAULT]",
		))
	}

	err = uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		coreResp, coreErr := uc.subscription.executeCore(txCtx, req, enriched)
		if coreErr != nil {
			return coreErr
		}
		resp = coreResp
		// At this point the subscription row is inserted (still inside the tx);
		// recognizePoolFunc reads it + the bound PricePlan + writes
		// Revenue + RevenueLineItem inside the SAME tx. Any failure rolls
		// the whole thing back — codex CRIT-5 atomicity contract.
		if invErr := uc.recognizePoolFunc.RecognizeAdHocPool(txCtx, enriched.GetId()); invErr != nil {
			return invErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Post-commit best-effort job instantiation — mirrors CreateSubscription.Execute.
	if uc.subscription.services.JobTemplateInstantiator != nil && pricePlan != nil {
		wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
		spawnJobs := true
		if override, set := contextutil.ExtractSpawnJobsOverride(ctx); set {
			spawnJobs = override
		}
		_ = uc.subscription.services.JobTemplateInstantiator.InstantiateJobsFromPlan(
			ctx, pricePlan.PlanId, enriched.ClientId, enriched.Id, wsID, spawnJobs,
		)
		// Job-instantiation failures stay non-fatal here too (matches plain path).
	}

	return resp, nil
}

// isAdHocPool returns true when the bound PricePlan is the codex CRIT-5
// target combo (AD_HOC × TOTAL_PACKAGE). Mirrors centymo's view-side predicate.
func isAdHocPool(pp *priceplanpb.PricePlan) bool {
	if pp == nil {
		return false
	}
	return pp.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_AD_HOC &&
		pp.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE
}
