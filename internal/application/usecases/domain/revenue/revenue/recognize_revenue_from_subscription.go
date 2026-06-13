package revenue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"google.golang.org/protobuf/proto"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

const (
	entitySubscription = "subscription"

	// Treatment tokens used for the preview badge — kept lowercase so the
	// drawer's lyngua key map is straightforward.
	treatmentRecurring  = "recurring"
	treatmentFirstCycle = "first_cycle"
	treatmentUsageBased = "usage_based"
	treatmentOneTime    = "one_time"
)

// RecognizeRevenueFromSubscriptionRepositories aggregates every cross-domain
// repository the use case needs. The pattern mirrors CreateRevenueUseCase but
// fans out further because revenue recognition reads from subscription, price
// plan, and product-price-plan domains.
type RecognizeRevenueFromSubscriptionRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
	PaymentTerm      paymenttermpb.PaymentTermDomainServiceServer

	// Milestone-billing branch (Phase C — milestone-billing plan §3).
	// Optional — when nil, MILESTONE plans are rejected with a clear error.
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
}

// RecognizeRevenueFromSubscriptionServices groups all business service
// dependencies. Mirrors CreateRevenueServices.
type RecognizeRevenueFromSubscriptionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator

	// 2026-04-30 cyclic-subscription-jobs plan §5.2 — piggyback hook.
	//
	// When set, the use case calls this AFTER successful revenue
	// recognition for cyclic plans, to materialise the missing cycle Job
	// (idempotent — returns existing if already present).
	//
	// Failure semantics — locked-in critical decision (plan §5.2):
	//   The piggyback is NON-FATAL. Failure does NOT roll back the
	//   recognized Revenue. Instead, the use case appends a structured
	//   warning to response.Warnings with shape:
	//       "cycle_job_spawn_failed: <err.Error()>"
	//
	//   This is the OPPOSITE of MaterializeJobsForSubscription's atomic-
	//   rollback semantics for the engagement shell. Reasoning:
	//   revenue recognition is the source of financial truth; cycle Jobs
	//   are operational housekeeping. Failing to spawn the cycle Job
	//   because (e.g.) the partial unique index briefly blocked it is
	//   recoverable later via the manual "Spawn this cycle now" action.
	//   Failing the WHOLE revenue recognition because of a downstream
	//   housekeeping error would be wrong.
	//
	// nil-safe: when unset, the piggyback is skipped entirely (no warning).
	MaterializeInstanceJobsForSubscription MaterializeInstanceJobsForSubscriptionInvoker

	// ComputeTaxes wires the post-recognize tax-compute hook
	// (tax-integration plan §4 Phase C). Optional — when nil the tax-compute
	// step is skipped entirely (no warning). Failure is non-fatal; any error
	// is appended as a structured "tax_compute_failed: <err>" warning so that
	// the recognized Revenue is never rolled back on account of tax bookkeeping.
	//
	// 2026-05-20 Plan 2 / Q-SDM-TAX — the composition root now satisfies this
	// Invoker by resolving the service-driven tax wrapper
	// (`usecases/service/tax`.ComputeTaxesForRevenueUseCase) via the dynamic
	// registry (servicetax.From). The previous wiring passed the entity-layer
	// use case directly; the new path routes through the proto-shaped wrapper
	// so the recognize flow consumes the formalized service contract
	// (`proto/v1/service/tax/compute.proto`) without re-importing the
	// entity-layer package. Failure semantics are unchanged — the wrapper's
	// ExecuteForRevenue is a thin pass-through to the entity compute.
	ComputeTaxes ComputeTaxesForRevenueInvoker
}

// MaterializeInstanceJobsForSubscriptionInvoker is the narrow contract for
// the recognize-piggyback hook. The concrete use case
// (subscription/subscription.MaterializeInstanceJobsForSubscriptionUseCase)
// is wired by the composition layer; the interface here keeps espyna's
// revenue package free of a cross-domain import cycle.
//
// Returning a non-nil error triggers the non-fatal warning shape — the
// piggyback failure is documented inline at the call site.
type MaterializeInstanceJobsForSubscriptionInvoker interface {
	Execute(ctx context.Context, subscriptionID, cyclePeriodStart string) error
}

// ComputeTaxesForRevenueInvoker is the narrow contract for the post-recognize
// tax-compute hook. Using a minimal interface allows the composition layer to
// wire the concrete tax use case without creating a cross-package import cycle
// and without requiring the recognize file to import the entire tax package.
//
// Failure semantics: non-fatal. Errors append a structured warning of the form
// "tax_compute_failed: <err>". The recognized Revenue is NOT rolled back.
// Rationale: revenue recognition is the financial source of truth; tax-line
// creation is bookkeeping that can be retried via the RecomputeTaxes admin
// action if the initial compute fails.
type ComputeTaxesForRevenueInvoker interface {
	ExecuteForRevenue(ctx context.Context, revenueID, workspaceID string) error
}

// RecognizeRevenueFromSubscriptionUseCase materializes a Revenue + N
// RevenueLineItems from a subscription's price-plan structure. See plan §3
// (docs/plan/20260425-subscription-revenue-recognition/plan.md) for the full
// algorithm.
type RecognizeRevenueFromSubscriptionUseCase struct {
	repositories RecognizeRevenueFromSubscriptionRepositories
	services     RecognizeRevenueFromSubscriptionServices
}

// NewRecognizeRevenueFromSubscriptionUseCase wires the use case.
func NewRecognizeRevenueFromSubscriptionUseCase(
	repositories RecognizeRevenueFromSubscriptionRepositories,
	services RecognizeRevenueFromSubscriptionServices,
) *RecognizeRevenueFromSubscriptionUseCase {
	return &RecognizeRevenueFromSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// SetMaterializeInstanceJobsForSubscription installs the recognize-piggyback
// invoker after construction. Used by the composition layer to wire the
// cyclic-instance-Job spawn hook (cyclic-subscription-jobs plan §5.2)
// without threading the dependency through the entire revenue.NewUseCases
// signature.
//
// Safe to call with nil — disables the piggyback (no warning, no spawn).
func (uc *RecognizeRevenueFromSubscriptionUseCase) SetMaterializeInstanceJobsForSubscription(
	invoker MaterializeInstanceJobsForSubscriptionInvoker,
) {
	if uc == nil {
		return
	}
	uc.services.MaterializeInstanceJobsForSubscription = invoker
}

// SetComputeTaxes installs the post-recognize tax-compute invoker after
// construction. Used by the composition layer (tax-integration plan §4 Phase C)
// to break the initialization ordering cycle: revenue is initialized before tax,
// so the tax use case is injected via a setter once both are ready.
//
// Safe to call with nil — disables the tax-compute hook (no warning, no lines).
func (uc *RecognizeRevenueFromSubscriptionUseCase) SetComputeTaxes(
	invoker ComputeTaxesForRevenueInvoker,
) {
	if uc == nil {
		return
	}
	uc.services.ComputeTaxes = invoker
}

// Execute orchestrates the revenue recognition flow. The shape of the request
// matches the CreateRevenueWithLineItems RPC; when dry_run is set the use case
// returns a preview without writing.
func (uc *RecognizeRevenueFromSubscriptionUseCase) Execute(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenue,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySubscription,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.validation.request_required",
			"Request is required [DEFAULT]",
		))
	}

	subscriptionID := req.GetSubscriptionId()
	if subscriptionID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.validation.subscription_id_required",
			"subscription_id is required [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil &&
		uc.services.Transactor.SupportsTransactions() &&
		!req.GetDryRun() {
		var result *revenuepb.CreateRevenueWithLineItemsResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, execErr := uc.executeCore(txCtx, req)
			if execErr != nil {
				return fmt.Errorf("revenue recognition failed: %w", execErr)
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

// debugTimingEnabled returns true when REVENUE_DEBUG_TIMING=1 in the process
// environment. Used to gate verbose per-phase timing logs in executeCore so the
// instrumentation can be flipped on live without re-instrumenting the file.
func debugTimingEnabled() bool {
	return os.Getenv("REVENUE_DEBUG_TIMING") == "1"
}

// executeCore is the transaction-agnostic body. It runs all reads, builds the
// preview, and (unless dry_run) writes the Revenue + lines.
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeCore(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	subscriptionID := req.GetSubscriptionId()

	// Phase timing — gated behind REVENUE_DEBUG_TIMING=1 so prod logs stay clean.
	// Set the env var before triggering a Run Invoices to capture per-phase costs.
	debug := debugTimingEnabled()
	t0 := time.Now()
	tPhase := t0
	logPhase := func(name string) {
		if debug {
			log.Printf("recognize: phase=%s took=%v", name, time.Since(tPhase))
			tPhase = time.Now()
		}
	}
	if debug {
		defer func() { log.Printf("recognize: total=%v sub=%s", time.Since(t0), subscriptionID) }()
	}

	// 1. Resolve subscription
	sub, err := uc.readSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	logPhase("resolve_subscription")
	if !sub.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"subscription.errors.subscription_inactive",
			"Subscription is inactive [DEFAULT]",
		))
	}

	// 2. Resolve price plan
	pricePlanID := sub.GetPricePlanId()
	if pricePlanID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"subscription.errors.price_plan_required",
			"Subscription has no price plan [DEFAULT]",
		))
	}
	pricePlan, err := uc.readPricePlan(ctx, pricePlanID)
	if err != nil {
		return nil, err
	}
	logPhase("resolve_price_plan")

	// 2a. Normalize incoherent kind × basis cells per plan §3.6 / followups §1.
	// Mutates the in-memory pointer only — does NOT write back to the DB.
	normalizePricePlan(pricePlan)

	// 2b. MILESTONE pre-branch normalization (milestone-billing plan §3).
	// MILESTONE plans never carry billing_cycle_*; the drawer clears it but we
	// defend in the use case in case a legacy or seeded row leaks through.
	if pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_MILESTONE {
		pricePlan.BillingCycleValue = nil
		pricePlan.BillingCycleUnit = nil
	}

	// 2c-pre. AD_HOC dispatch — runs BEFORE the milestone-only billing_event_id
	// rejection (codex CRIT-2). AD_HOC × PER_OCCURRENCE legitimately uses
	// billing_event_id, so it must be evaluated above that switch.
	// See docs/plan/20260501-ad-hoc-subscription-billing/plan.md §4.2.
	billingEventID := strings.TrimSpace(req.GetBillingEventId())
	if pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_AD_HOC {
		// Currency + client-drift checks run here so AD_HOC inherits them.
		client := uc.readClient(ctx, sub.GetClientId())
		priceSchedule := uc.readPriceSchedule(ctx, pricePlan.GetPriceScheduleId())
		planCurrency := pricePlan.GetBillingCurrency()
		clientCurrency := ""
		if client != nil {
			clientCurrency = client.GetBillingCurrency()
		}
		if clientCurrency != "" && planCurrency != "" && clientCurrency != planCurrency {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"revenue.errors.currency_mismatch",
				"Client billing currency does not match the rate card [DEFAULT]",
			))
		}
		return uc.executeAdHoc(ctx, req, sub, pricePlan, priceSchedule, client, planCurrency, billingEventID)
	}

	// 2c. Reject mismatched billing_event_id × billing_kind combos before any
	// other work. Both checks here keep the rest of the file's logic
	// candidate-agnostic — non-MILESTONE branches never see milestone fields.
	switch {
	case pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_MILESTONE:
		if billingEventID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"revenue.errors.milestone_required",
				"A billing event is required for milestone plans [DEFAULT]",
			))
		}
	case billingEventID != "":
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.milestone_not_applicable",
			"Billing event id is only valid on milestone plans [DEFAULT]",
		))
	}

	// 3. Hydrate context (client, payment term, price schedule)
	client := uc.readClient(ctx, sub.GetClientId())
	priceSchedule := uc.readPriceSchedule(ctx, pricePlan.GetPriceScheduleId())
	logPhase("resolve_client_priceschedule")

	// 4. Currency assertion (hard block per plan §11.4)
	planCurrency := pricePlan.GetBillingCurrency()
	clientCurrency := ""
	if client != nil {
		clientCurrency = client.GetBillingCurrency()
	}
	if clientCurrency != "" && planCurrency != "" && clientCurrency != planCurrency {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.currency_mismatch",
			"Client billing currency does not match the rate card [DEFAULT]",
		))
	}

	// §3.6 (20260427-plan-client-scope) — subscription/plan client drift guard.
	// resolvedClientID = pricePlan.Plan.client_id when the join is present,
	// else fall back to the denormalized mirror on PricePlan itself. Reject
	// when both sides are non-empty and disagree — protects revenue
	// recognition from a desynced client identity (e.g. direct DB writes).
	resolvedClientID := pricePlan.GetClientId()
	if joinedPlan := pricePlan.GetPlan(); joinedPlan != nil {
		if joinedPlan.GetClientId() != "" {
			resolvedClientID = joinedPlan.GetClientId()
		}
	}
	if subClientID := sub.GetClientId(); subClientID != "" && resolvedClientID != "" && subClientID != resolvedClientID {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.subscriptionPlanClientDrift",
			"Subscription and price plan belong to different clients — recognition blocked. [DEFAULT]",
		))
	}

	// 4a. MILESTONE branch — entirely separate write path. The remainder of
	// this function is unchanged for ONE_TIME / RECURRING / CONTRACT plans.
	if pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_MILESTONE {
		return uc.executeMilestone(ctx, req, sub, pricePlan, priceSchedule, client, planCurrency)
	}

	// 5. List ProductPricePlans for this PricePlan
	ppps := uc.listProductPricePlans(ctx, pricePlanID)
	logPhase("list_product_price_plans")

	// 6. Idempotency + first-cycle detection require existing revenue rows
	priorRevenues := uc.listRevenuesForSubscription(ctx, subscriptionID)
	logPhase("list_prior_revenues")
	periodStart := strings.TrimSpace(req.GetPeriodStart())
	periodEnd := strings.TrimSpace(req.GetPeriodEnd())

	// 6a. Idempotency check (skip on skipHeader since the caller already created
	// the Revenue header — no period collision possible against itself)
	if !req.GetSkipHeader() && periodStart != "" && periodEnd != "" {
		if conflictID := findIdempotencyConflict(priorRevenues, periodStart, periodEnd); conflictID != "" {
			resp := &revenuepb.CreateRevenueWithLineItemsResponse{
				Success:              false,
				ConflictingRevenueId: stringPtrLocal(conflictID),
			}
			errPb := &commonpb.Error{
				Code: "period_already_invoiced",
				Message: contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.Translator,
					"revenue.errors.period_already_invoiced",
					"An invoice already exists for this period [DEFAULT]",
				),
			}
			resp.Error = errPb
			return resp, errors.New(errPb.Message)
		}
	}

	// 7. Build first-cycle PPP-id set (any PPP that already has a non-cancelled
	// line on a prior revenue is "not first cycle" anymore).
	priorLinesByPPP := uc.collectPriorLinePPPIDs(ctx, priorRevenues)
	logPhase("collect_prior_line_ppps")

	// 8. Build line items per plan §3.3
	var warnings []string
	overridesByPPP := indexOverrides(req.GetOverrides())
	lines, treatments, lineWarnings := buildLineItems(pricePlan, ppps, overridesByPPP, priorLinesByPPP)
	warnings = append(warnings, lineWarnings...)
	logPhase("build_line_items")

	// 9. Empty-PPP fallback for TOTAL_PACKAGE plans (edge case 9 in plan)
	if len(lines) == 0 && pricePlan.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		lines, treatments = buildBundleLine(pricePlan)
	}

	// 9a. Empty-line guard (followups §2). Outside the TOTAL_PACKAGE fallback,
	// a zero-line Revenue is never legitimate — reject so callers (drawer,
	// skip-header autoPopulate, future schedulers) cannot accidentally write
	// an unbillable invoice.
	if len(lines) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.no_lines_to_invoice",
			"Cannot create an invoice with no line items [DEFAULT]",
		))
	}

	// 10. Build header per plan §3.4
	revenueDate := strings.TrimSpace(req.GetRevenueDate())
	if revenueDate == "" {
		revenueDate = time.Now().UTC().Format("2006-01-02")
	}

	// Compute total based on amount basis.
	var totalAmount int64
	for _, l := range lines {
		totalAmount += l.GetTotalPrice()
	}
	if pricePlan.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE &&
		pricePlan.GetBillingKind() == priceplanpb.BillingKind_BILLING_KIND_ONE_TIME {
		// For a TOTAL_PACKAGE one-time plan, header total takes precedence over
		// the sum when no PPPs are present; otherwise sum wins (operator overrides).
		if len(ppps) == 0 {
			totalAmount = pricePlan.GetBillingAmount()
		}
	}

	// Build preview lines (always populated — drawer renders from these).
	previewLines := buildPreviewLines(lines, treatments)

	// Dry-run path: never write, return the preview only.
	if req.GetDryRun() {
		return &revenuepb.CreateRevenueWithLineItemsResponse{
			Success:      true,
			PreviewLines: previewLines,
			Warnings:     warnings,
		}, nil
	}

	// Skip-header path: caller has already inserted the Revenue; only the line
	// items need to be written. Used by the manual revenue-add flow.
	if req.GetSkipHeader() {
		existingRevenueID := req.GetExistingRevenueId()
		if existingRevenueID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"revenue.validation.existing_revenue_id_required",
				"existing_revenue_id is required when skip_header is true [DEFAULT]",
			))
		}
		if err := uc.persistLineItems(ctx, existingRevenueID, lines); err != nil {
			return nil, err
		}
		return &revenuepb.CreateRevenueWithLineItemsResponse{
			Success:      true,
			PreviewLines: previewLines,
			Warnings:     warnings,
		}, nil
	}

	// Standard path: insert header + lines atomically.
	header := uc.buildHeader(req, sub, pricePlan, priceSchedule, client, planCurrency,
		periodStart, periodEnd, revenueDate, totalAmount, warnings)
	logPhase("build_header")

	createdRevenue, err := uc.persistRevenue(ctx, header)
	if err != nil {
		// DB-level race: the partial unique index on (subscription_id,
		// period_marker) caught a concurrent insert. Re-list and surface the
		// same conflict shape the read-time idempotency check uses so the
		// drawer renders one consistent banner.
		if strings.Contains(err.Error(), "period_already_invoiced") &&
			periodStart != "" && periodEnd != "" {
			fresh := uc.listRevenuesForSubscription(ctx, subscriptionID)
			if conflictID := findIdempotencyConflict(fresh, periodStart, periodEnd); conflictID != "" {
				resp := &revenuepb.CreateRevenueWithLineItemsResponse{
					Success:              false,
					ConflictingRevenueId: stringPtrLocal(conflictID),
					Error: &commonpb.Error{
						Code: "period_already_invoiced",
						Message: contextutil.GetTranslatedMessageWithContext(
							ctx, uc.services.Translator,
							"revenue.errors.period_already_invoiced",
							"An invoice already exists for this period [DEFAULT]",
						),
					},
				}
				return resp, errors.New(resp.GetError().GetMessage())
			}
		}
		return nil, err
	}
	logPhase("persist_revenue")

	if err := uc.persistLineItems(ctx, createdRevenue.GetId(), lines); err != nil {
		return nil, err
	}
	logPhase("persist_line_items")

	// 2026-05-10 tax-integration plan §4 Phase C — post-recognize tax compute.
	//
	// AFTER successful Revenue + RevenueLineItem persistence, fire the tax
	// compute step. This is INTENTIONALLY non-fatal: any error is appended as
	// a structured "tax_compute_failed: <err>" warning and execution continues.
	// Revenue recognition is the source of financial truth; RevenueTaxLine rows
	// are bookkeeping that can be retried via the RecomputeTaxes admin action.
	//
	// workspace_id: sourced from sub.GetWorkspaceId(). The Revenue proto has no
	// workspace_id field; the subscription always carries the tenant identifier.
	if uc.services.ComputeTaxes != nil {
		wsID := sub.GetWorkspaceId()
		if computeErr := uc.services.ComputeTaxes.ExecuteForRevenue(ctx, createdRevenue.GetId(), wsID); computeErr != nil {
			warnings = append(warnings,
				fmt.Sprintf("tax_compute_failed: %s", computeErr.Error()))
		}
		logPhase("compute_taxes")
	}

	// 2026-04-30 cyclic-subscription-jobs plan §5.2 — recognize-piggyback.
	//
	// AFTER successful Revenue + RevenueLineItem persistence, fire the
	// cycle-Job spawn for cyclic plans. This is INTENTIONALLY non-fatal:
	// any error is appended as a structured warning and execution continues.
	// Revenue recognition is the source of financial truth; cycle Jobs are
	// operational housekeeping. See MaterializeInstanceJobsForSubscription
	// service field doc for the full reasoning.
	//
	// Idempotency: MaterializeInstanceJobsForSubscription is idempotent —
	// if the cycle Job already exists for this period_start, the call
	// returns the existing row without an error.
	//
	// Period normalization: the recognize-revenue drawer submits an RFC3339
	// timestamp on `period_start` (the form's date+time grid is stitched into
	// an ISO string by the centymo handler). The instance-jobs use case
	// expects YYYY-MM-DD because cycle math is day-grained (`time.Parse
	// "2006-01-02"` in computeCycleEnd). Strip the time portion here so the
	// piggyback fires with a clean date — otherwise computeCycleEnd would
	// reject the input and the warning would silently swallow a config bug.
	if uc.services.MaterializeInstanceJobsForSubscription != nil && IsCyclicPricePlan(pricePlan) {
		cyclePeriodStart := normalizePeriodStartForCycleSpawn(periodStart)
		if err := uc.services.MaterializeInstanceJobsForSubscription.Execute(
			ctx, sub.GetId(), cyclePeriodStart,
		); err != nil {
			warnings = append(warnings,
				fmt.Sprintf("cycle_job_spawn_failed: %s", err.Error()))
		}
	}

	return &revenuepb.CreateRevenueWithLineItemsResponse{
		Success:      true,
		Data:         []*revenuepb.Revenue{createdRevenue},
		PreviewLines: previewLines,
		Warnings:     warnings,
	}, nil
}

// normalizePeriodStartForCycleSpawn coerces the recognize-revenue drawer's
// period_start input into a YYYY-MM-DD date that the instance-jobs use case
// can parse with `time.Parse("2006-01-02", …)`. The drawer submits an
// RFC3339 timestamp (date+time grid stitched into ISO format) but cycle math
// is day-grained — passing the full RFC3339 through would trigger a parse
// error inside computeCycleEnd and surface as a non-fatal warning, masking
// the cycle-Job spawn failure that drives Operations-tab visibility.
//
// Empty input is preserved (the use case reads it as "spawn the next
// un-spawned cycle"). Non-parseable input is also passed through unchanged
// so the instance-jobs use case still emits the original parse error
// verbatim — easier to diagnose downstream than a silently-swallowed value.
func normalizePeriodStartForCycleSpawn(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Already a YYYY-MM-DD date — fast path.
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return s
	}
	// RFC3339 (with offset). The drawer's hidden ISO field uses this shape.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02")
	}
	// RFC3339Nano fallback in case any caller adds sub-second precision.
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Format("2006-01-02")
	}
	// Datetime-local without timezone (`2006-01-02T15:04:05`).
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.Format("2006-01-02")
	}
	return s
}

// IsCyclicPricePlan returns true when the PricePlan participates in the
// two-tier engagement+instance Job model (cyclic-subscription-jobs plan §3.1).
//
// Mirrors subscription.eligibleForInstanceSpawn so the recognize-revenue
// package doesn't import the subscription package directly. The AD_HOC plan
// (downstream) extends both predicates symmetrically.
func IsCyclicPricePlan(pp *priceplanpb.PricePlan) bool {
	if pp == nil {
		return false
	}
	kind := pp.GetBillingKind()
	if kind == priceplanpb.BillingKind_BILLING_KIND_RECURRING {
		return true
	}
	if kind == priceplanpb.BillingKind_BILLING_KIND_CONTRACT && pp.GetBillingCycleValue() > 0 {
		return true
	}
	return false
}

// readSubscription wraps the subscription RPC for clarity.
func (uc *RecognizeRevenueFromSubscriptionUseCase) readSubscription(
	ctx context.Context, id string,
) (*subscriptionpb.Subscription, error) {
	if uc.repositories.Subscription == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"subscription.errors.repository_unavailable",
			"Subscription repository is not configured [DEFAULT]",
		))
	}
	resp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf("read subscription: %w", err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"subscription.errors.not_found",
			"Subscription not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *RecognizeRevenueFromSubscriptionUseCase) readPricePlan(
	ctx context.Context, id string,
) (*priceplanpb.PricePlan, error) {
	if uc.repositories.PricePlan == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"price_plan.errors.repository_unavailable",
			"Price plan repository is not configured [DEFAULT]",
		))
	}
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf("read price plan: %w", err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"price_plan.errors.not_found",
			"Price plan not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

// readClient is best-effort — when nil/error, returns nil and the use case
// proceeds without currency assertion.
func (uc *RecognizeRevenueFromSubscriptionUseCase) readClient(ctx context.Context, id string) *clientpb.Client {
	if id == "" || uc.repositories.Client == nil {
		return nil
	}
	resp, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
		Data: &clientpb.Client{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil
	}
	return resp.GetData()[0]
}

// readPriceSchedule is informational — used to source location_id when present.
func (uc *RecognizeRevenueFromSubscriptionUseCase) readPriceSchedule(
	ctx context.Context, id string,
) *priceschedulepb.PriceSchedule {
	if id == "" || uc.repositories.PriceSchedule == nil {
		return nil
	}
	resp, err := uc.repositories.PriceSchedule.ReadPriceSchedule(ctx, &priceschedulepb.ReadPriceScheduleRequest{
		Data: &priceschedulepb.PriceSchedule{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil
	}
	return resp.GetData()[0]
}

// listProductPricePlans returns the PPPs filtered to the given price plan.
//
// 2026-05-11 perf fix: previously this call passed an empty request, pulling
// every PPP row in the workspace and filtering in Go. With even a moderate-
// sized workspace (dozens of plans × several products each) that's hundreds
// of rows materialized + protojson-unmarshalled per recognize call — the
// dominant cost in Run Invoices for single-period runs. The postgres adapter
// honors `req.Filters` (commonpb.FilterRequest, see ListProductPricePlans in
// contrib/postgres/internal/adapter/subscription/product_price_plan.go), so
// pushing the price_plan_id predicate to SQL collapses the read to the
// handful of rows we actually need. Falls back to the original full-list +
// Go-side filter only when filtering returns an error — keeps the path
// resilient to adapter providers that may not honor filters (mock_db, etc.).
func (uc *RecognizeRevenueFromSubscriptionUseCase) listProductPricePlans(
	ctx context.Context, pricePlanID string,
) []*productpriceplanpb.ProductPricePlan {
	if uc.repositories.ProductPricePlan == nil {
		return nil
	}
	filtered, err := uc.repositories.ProductPricePlan.ListProductPricePlans(
		ctx, &productpriceplanpb.ListProductPricePlansRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "price_plan_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    pricePlanID,
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
				},
			},
		},
	)
	if err == nil && filtered != nil {
		// Defensive: some adapters may silently ignore unknown filter fields and
		// return every row. Re-filter in Go to guarantee correctness regardless
		// of provider behavior. Cheap when the SQL filter worked (list already
		// small) and necessary when it didn't.
		data := filtered.GetData()
		out := make([]*productpriceplanpb.ProductPricePlan, 0, len(data))
		for _, ppp := range data {
			if ppp.GetPricePlanId() == pricePlanID {
				out = append(out, ppp)
			}
		}
		return out
	}

	// Fallback: full-list + Go-side filter for providers that don't accept the
	// price_plan_id predicate. Matches the pre-fix behavior so functionality is
	// preserved even if the filter shape is unsupported.
	resp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(
		ctx, &productpriceplanpb.ListProductPricePlansRequest{},
	)
	if err != nil || resp == nil {
		return nil
	}
	out := make([]*productpriceplanpb.ProductPricePlan, 0, len(resp.GetData()))
	for _, ppp := range resp.GetData() {
		if ppp.GetPricePlanId() == pricePlanID {
			out = append(out, ppp)
		}
	}
	return out
}

// listRevenuesForSubscription pulls all prior revenues for the subscription.
// Returns nil on error (treats as "no prior revenues" — same as a brand-new
// subscription).
func (uc *RecognizeRevenueFromSubscriptionUseCase) listRevenuesForSubscription(
	ctx context.Context, subscriptionID string,
) []*revenuepb.Revenue {
	if uc.repositories.Revenue == nil {
		return nil
	}
	resp, err := uc.repositories.Revenue.ListRevenues(ctx, &revenuepb.ListRevenuesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "subscription_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    subscriptionID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetData()
}

// collectPriorLinePPPIDs returns the set of product_price_plan_ids that already
// appear on at least one non-cancelled prior revenue's line items. Used for
// first-cycle detection: a PPP in this set is no longer "first cycle".
func (uc *RecognizeRevenueFromSubscriptionUseCase) collectPriorLinePPPIDs(
	ctx context.Context, priorRevenues []*revenuepb.Revenue,
) map[string]bool {
	out := make(map[string]bool)
	if uc.repositories.RevenueLineItem == nil {
		return out
	}
	for _, rev := range priorRevenues {
		if rev.GetStatus() == "cancelled" {
			continue
		}
		resp, err := uc.repositories.RevenueLineItem.ListRevenueLineItems(
			ctx, &revenuelineitempb.ListRevenueLineItemsRequest{
				RevenueId: stringPtrLocal(rev.GetId()),
			},
		)
		if err != nil || resp == nil {
			continue
		}
		for _, li := range resp.GetData() {
			if pppID := li.GetProductPricePlanId(); pppID != "" {
				out[pppID] = true
			}
		}
	}
	return out
}

// normalizePricePlan coerces incoherent kind × basis cells and clears fields
// that have no meaning under the resulting combination. Mirrors plan §3.6 of
// the parent revenue-recognition plan; called once per Execute so the rest of
// the engine can rely on a coherent in-memory PricePlan.
//
// The mutation is local to the in-memory pointer the use case operates on —
// nothing here writes back to the database. The drawer's default-period
// calculation reads the same pointer, so cycle/term clearing keeps the drawer
// and engine in sync on legacy rows.
func normalizePricePlan(pp *priceplanpb.PricePlan) {
	if pp == nil {
		return
	}
	kind := pp.GetBillingKind()
	basis := pp.GetAmountBasis()

	// Coerce incoherent cells.
	if kind == priceplanpb.BillingKind_BILLING_KIND_ONE_TIME &&
		basis == priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE {
		coerced := priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE
		pp.AmountBasis = coerced
		basis = coerced
	}
	if kind == priceplanpb.BillingKind_BILLING_KIND_RECURRING &&
		basis == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		coerced := priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE
		pp.AmountBasis = coerced
		basis = coerced
	}
	// Per scenarios.md §2.7 row "CONTRACT | TOTAL_PACKAGE | Treated as ONE_TIME":
	// the contract's term IS the period, so a TOTAL_PACKAGE basis charges the
	// header amount once up-front. Coerce kind → ONE_TIME so buildLineItems
	// stamps `treatment=one_time` and downstream cycle/term logic runs the
	// ONE_TIME path. Mutation is in-memory only — DB row is unchanged.
	if kind == priceplanpb.BillingKind_BILLING_KIND_CONTRACT &&
		basis == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		pp.BillingKind = priceplanpb.BillingKind_BILLING_KIND_ONE_TIME
		kind = priceplanpb.BillingKind_BILLING_KIND_ONE_TIME
	}

	// Clear billing_cycle_* when the cell has no cadence.
	if kind == priceplanpb.BillingKind_BILLING_KIND_ONE_TIME ||
		basis != priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE {
		pp.BillingCycleValue = nil
		pp.BillingCycleUnit = nil
	}

	// Clear default_term_* on open-ended recurring (kind=RECURRING, basis=PER_CYCLE).
	if kind == priceplanpb.BillingKind_BILLING_KIND_RECURRING &&
		basis == priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE {
		pp.DefaultTermValue = nil
		pp.DefaultTermUnit = nil
	}
}

// buildLineItems materializes the line items for the recognition.
//
// Returns:
//
//	lines — the concrete RevenueLineItem rows to insert (without Id / RevenueId)
//	treatments — parallel slice of treatment tokens for preview badges
//	warnings — any non-blocking notices to surface in the drawer
func buildLineItems(
	pricePlan *priceplanpb.PricePlan,
	ppps []*productpriceplanpb.ProductPricePlan,
	overridesByPPP map[string]*revenuepb.LineItemOverride,
	priorPPPs map[string]bool,
) (lines []*revenuelineitempb.RevenueLineItem, treatments []string, warnings []string) {
	billingKind := pricePlan.GetBillingKind()
	currency := pricePlan.GetBillingCurrency()

	for _, ppp := range ppps {
		// Operator removed this line in the preview.
		if ov := overridesByPPP[ppp.GetId()]; ov != nil && ov.GetRemoved() {
			continue
		}

		var treatment string
		switch billingKind {
		case priceplanpb.BillingKind_BILLING_KIND_ONE_TIME:
			// billing_treatment is ignored — every line charges once
			treatment = treatmentOneTime
		case priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			priceplanpb.BillingKind_BILLING_KIND_CONTRACT,
			priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED:
			switch ppp.GetBillingTreatment() {
			case productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED:
				warnings = append(warnings,
					fmt.Sprintf("usage-based line %q skipped — record via metering",
						describeLine(ppp)))
				continue
			case productpriceplanpb.BillingTreatment_BILLING_TREATMENT_ONE_TIME_INITIAL:
				if priorPPPs[ppp.GetId()] {
					// Already invoiced on a prior cycle — suppress.
					continue
				}
				treatment = treatmentFirstCycle
			default: // RECURRING + UNSPECIFIED legacy default
				treatment = treatmentRecurring
			}
		default:
			treatment = treatmentRecurring
		}

		unitPrice := ppp.GetBillingAmount()
		quantity := 1.0
		description := describeLine(ppp)
		if ov := overridesByPPP[ppp.GetId()]; ov != nil {
			if ov.UnitPrice != nil {
				unitPrice = *ov.UnitPrice
			}
			if ov.Quantity != nil {
				quantity = *ov.Quantity
			}
			if ov.Description != nil && *ov.Description != "" {
				description = *ov.Description
			}
		}

		totalPrice := int64(float64(unitPrice) * quantity)
		pppID := ppp.GetId()
		line := &revenuelineitempb.RevenueLineItem{
			Description: description,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
			// LineAmount mirrors TotalPrice — this is the field ComputeTaxesForRevenue
			// reads for tax base calculation (plan §4 Phase C2 fix).
			LineAmount:         totalPrice,
			LineItemType:       "item",
			ProductPricePlanId: stringPtrLocal(pppID),
		}
		_ = currency // currency lives on the Revenue header — RevenueLineItem proto has no currency field

		// Populate tax snapshots from the ProductPricePlan → Product join chain.
		// Resolution order: PPP override → Product (plan §4 Phase C3).
		// IDs (TaxTreatmentId, WithholdingClassId) are FK references; the snapshot
		// stores the code string (e.g. "STANDARD", "PROFESSIONAL_CORPORATE").
		// When the PPP adapter returns a joined Product, extract the IDs so that
		// ComputeTaxesForRevenue's product-fallback path has the right product_id
		// to resolve the code. The product_id is already set above via ProductPricePlanId.
		//
		// NOTE: Resolving the CODE (not ID) requires a TaxTreatment/TaxClass repo read
		// which is not wired into this use case. The compute fallback path handles that:
		//   compute reads product.GetTaxTreatmentId() → readTaxTreatment → code
		// This is correct when TaxTreatmentSnapshot is empty (compute falls through to product).
		// When the PPP's product join carries the codes directly (future enrichment), set them here.
		if prod := ppp.GetProductPlan().GetProduct(); prod != nil {
			// ProductId is already carried via ProductPricePlanId → compute resolves it.
			// If a future PPP enrichment adds TaxTreatment.code inline, set it here.
			_ = prod
		}

		lines = append(lines, line)
		treatments = append(treatments, treatment)
	}
	return lines, treatments, warnings
}

// buildBundleLine emits a single line representing a bundle/total-package plan
// when the PPP list is empty (edge case 9 in plan §6).
func buildBundleLine(
	pricePlan *priceplanpb.PricePlan,
) ([]*revenuelineitempb.RevenueLineItem, []string) {
	name := pricePlan.GetName()
	if name == "" {
		name = "Subscription"
	}
	amount := pricePlan.GetBillingAmount()
	line := &revenuelineitempb.RevenueLineItem{
		Description:  name,
		Quantity:     1,
		UnitPrice:    amount,
		TotalPrice:   amount,
		LineAmount:   amount, // Phase 4 C2: mirrors TotalPrice for tax base calculation.
		LineItemType: "item",
	}
	return []*revenuelineitempb.RevenueLineItem{line}, []string{treatmentOneTime}
}

// buildPreviewLines renders the preview slice from the (lines, treatments)
// pair. The drawer renders these directly — the live Revenue/RevenueLineItem
// rows are only inserted when dry_run = false.
func buildPreviewLines(
	lines []*revenuelineitempb.RevenueLineItem,
	treatments []string,
) []*revenuepb.PreviewLineItem {
	out := make([]*revenuepb.PreviewLineItem, 0, len(lines))
	for i, l := range lines {
		t := ""
		if i < len(treatments) {
			t = treatments[i]
		}
		ppp := ""
		if p := l.GetProductPricePlanId(); p != "" {
			ppp = p
		}
		out = append(out, &revenuepb.PreviewLineItem{
			ProductPricePlanId: ppp,
			Description:        l.GetDescription(),
			UnitPrice:          l.GetUnitPrice(),
			Quantity:           l.GetQuantity(),
			TotalPrice:         l.GetTotalPrice(),
			Treatment:          t,
		})
	}
	return out
}

// buildHeader assembles the Revenue header per plan §3.4. The caller is
// responsible for ID generation (handled inside persistRevenue).
func (uc *RecognizeRevenueFromSubscriptionUseCase) buildHeader(
	req *revenuepb.CreateRevenueWithLineItemsRequest,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	priceSchedule *priceschedulepb.PriceSchedule,
	client *clientpb.Client,
	planCurrency string,
	periodStart, periodEnd, revenueDate string,
	totalAmount int64,
	warnings []string,
) *revenuepb.Revenue {
	subID := sub.GetId()
	header := &revenuepb.Revenue{
		Name:           sub.GetName() + buildPeriodSuffix(periodStart, periodEnd),
		ClientId:       sub.GetClientId(),
		RevenueDate:    stringPtrLocal(revenueDate),
		TotalAmount:    totalAmount,
		Currency:       planCurrency,
		Status:         "draft",
		SubscriptionId: stringPtrLocal(subID),
		Notes:          stringPtrLocal(buildNotes(periodStart, periodEnd, warnings)),
	}
	// Inherit on the Revenue header.
	if priceSchedule != nil {
		header.LocationId = priceSchedule.GetLocationId()
	}
	if client != nil && client.GetPaymentTermId() != "" {
		ptID := client.GetPaymentTermId()
		header.PaymentTermId = &ptID
	}
	// Pass-through of operator-supplied data on req.Data — operator overrides
	// take precedence over the synthesized header values.
	if d := req.GetData(); d != nil {
		if d.GetReferenceNumber() != "" {
			ref := d.GetReferenceNumber()
			header.ReferenceNumber = &ref
		}
		if d.GetLocationId() != "" {
			header.LocationId = d.GetLocationId()
		}
		if d.GetNotes() != "" {
			combined := buildNotes(periodStart, periodEnd, warnings) + "\n\n" + d.GetNotes()
			header.Notes = stringPtrLocal(combined)
		}
		if d.GetPaymentTermId() != "" {
			pt := d.GetPaymentTermId()
			header.PaymentTermId = &pt
		}
		if d.GetRevenueCategoryId() != "" {
			rc := d.GetRevenueCategoryId()
			header.RevenueCategoryId = &rc
		}

		// Phase 4 H4 — tax + FX snapshot pass-through from the recognize drawer.
		// The drawer reads workspace settings and stamps these on req.Data before
		// submission. Passing them through here ensures RevenueTaxLine compute can
		// read the correct inclusive/enabled flags from the persisted Revenue.
		if d.TaxInclusivePricingSnapshot != nil {
			header.TaxInclusivePricingSnapshot = d.TaxInclusivePricingSnapshot
		}
		if d.TaxComputationEnabledSnapshot != nil {
			header.TaxComputationEnabledSnapshot = d.TaxComputationEnabledSnapshot
		}
		if d.GetBillingCurrency() != "" {
			bc := d.GetBillingCurrency()
			header.BillingCurrency = &bc
		}
		if d.GetForexRateMicroUnits() != 0 {
			fx := d.GetForexRateMicroUnits()
			header.ForexRateMicroUnits = &fx
		}
		if d.GetForexRateSource() != "" {
			frs := d.GetForexRateSource()
			header.ForexRateSource = &frs
		}
	}
	_ = pricePlan // kept for future use (e.g. embedding plan name into the header name)
	return header
}

// persistRevenue creates the revenue row, applying the same enrichment as
// CreateRevenueUseCase (ID, audit timestamps).
func (uc *RecognizeRevenueFromSubscriptionUseCase) persistRevenue(
	ctx context.Context, header *revenuepb.Revenue,
) (*revenuepb.Revenue, error) {
	now := time.Now()
	if header.Id == "" && uc.services.IDGenerator != nil {
		header.Id = uc.services.IDGenerator.GenerateID()
	}
	created := now.UnixMilli()
	createdStr := now.Format(time.RFC3339)
	header.DateCreated = &created
	header.DateCreatedString = &createdStr
	header.DateModified = &created
	header.DateModifiedString = &createdStr
	header.Active = true

	resp, err := uc.repositories.Revenue.CreateRevenue(ctx, &revenuepb.CreateRevenueRequest{
		Data: header,
	})
	if err != nil {
		return nil, fmt.Errorf("create revenue: %w", err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.create_failed",
			"Failed to create revenue [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

// persistLineItems writes each line item with revenue_id pointing at the just-
// created revenue.
func (uc *RecognizeRevenueFromSubscriptionUseCase) persistLineItems(
	ctx context.Context, revenueID string, lines []*revenuelineitempb.RevenueLineItem,
) error {
	if uc.repositories.RevenueLineItem == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue_line_item.errors.repository_unavailable",
			"Revenue line item repository is not configured [DEFAULT]",
		))
	}
	for _, l := range lines {
		if l.Id == "" && uc.services.IDGenerator != nil {
			l.Id = uc.services.IDGenerator.GenerateID()
		}
		l.RevenueId = revenueID
		l.Active = true
		if _, err := uc.repositories.RevenueLineItem.CreateRevenueLineItem(
			ctx, &revenuelineitempb.CreateRevenueLineItemRequest{Data: l},
		); err != nil {
			return fmt.Errorf("create revenue line item: %w", err)
		}
	}
	return nil
}

// findIdempotencyConflict returns the ID of an existing non-cancelled Revenue
// whose period (encoded in notes) matches the supplied (periodStart, periodEnd)
// pair. v1 stores period boundaries in notes — see plan §2.6.
func findIdempotencyConflict(
	priorRevenues []*revenuepb.Revenue, periodStart, periodEnd string,
) string {
	want := buildPeriodMarker(periodStart, periodEnd)
	for _, rev := range priorRevenues {
		if rev.GetStatus() == "cancelled" {
			continue
		}
		if strings.Contains(rev.GetNotes(), want) {
			return rev.GetId()
		}
	}
	return ""
}

// indexOverrides returns a (ppp_id → override) map for fast lookup.
func indexOverrides(overrides []*revenuepb.LineItemOverride) map[string]*revenuepb.LineItemOverride {
	out := make(map[string]*revenuepb.LineItemOverride, len(overrides))
	for _, ov := range overrides {
		if ov.GetProductPricePlanId() != "" {
			out[ov.GetProductPricePlanId()] = ov
		}
	}
	return out
}

// describeLine returns the best human-readable description for a PPP, falling
// back to the PPP id when the join returns no product/plan name.
func describeLine(ppp *productpriceplanpb.ProductPricePlan) string {
	if pp := ppp.GetProductPlan(); pp != nil {
		if p := pp.GetProduct(); p != nil {
			if name := p.GetName(); name != "" {
				return name
			}
		}
	}
	return ppp.GetId()
}

// buildPeriodSuffix builds a " — Period: YYYY-MM-DD → YYYY-MM-DD" suffix for
// the Revenue.name field. Returns "" when both bounds are empty.
func buildPeriodSuffix(start, end string) string {
	if start == "" && end == "" {
		return ""
	}
	return " — " + buildPeriodMarker(start, end)
}

// buildPeriodMarker is the canonical period encoding used across name + notes.
// Storing it identically in both fields keeps idempotency detection robust.
func buildPeriodMarker(start, end string) string {
	switch {
	case start != "" && end != "":
		return fmt.Sprintf("Period: %s → %s", start, end)
	case start != "":
		return fmt.Sprintf("Period: %s →", start)
	case end != "":
		return fmt.Sprintf("Period: → %s", end)
	default:
		return ""
	}
}

// buildNotes prefixes the notes with the period marker and any warnings.
func buildNotes(start, end string, warnings []string) string {
	var b strings.Builder
	if marker := buildPeriodMarker(start, end); marker != "" {
		b.WriteString(marker)
	}
	if len(warnings) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Warnings:\n")
		for _, w := range warnings {
			b.WriteString("- ")
			b.WriteString(w)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// stringPtrLocal is a tiny helper for the optional-string proto fields. Mirror
// of the stringPtr helper used elsewhere in the package.
func stringPtrLocal(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// MILESTONE branch (milestone-billing plan §3 + flow.md §11)
// ---------------------------------------------------------------------------

// executeMilestone handles BILLING_KIND_MILESTONE plans end-to-end:
//
//  1. Read BillingEvent, validate subscription match + status.
//  2. Idempotency: if a non-cancelled Revenue already references this event,
//     return conflicting_revenue_id (no second insert).
//  3. Resolve target amount (override or full).
//  4. Over-billing guard: sum of BillingEvents under the same template phase
//     for the same subscription cannot exceed the template-resolved total.
//  5. Build lines from PPP rows tagged with the same job_template_phase_id.
//  6. Atomic write: Revenue + RevenueLineItems + UpdateBillingEvent.
//  7. Optional child DEFERRED event when leave_remainder_open=true.
//
// Currency check has already happened in executeCore (shared across kinds).
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeMilestone(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	priceSchedule *priceschedulepb.PriceSchedule,
	client *clientpb.Client,
	planCurrency string,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	if uc.repositories.BillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_repository_unavailable",
			"Billing event repository is not configured [DEFAULT]",
		))
	}

	billingEventID := strings.TrimSpace(req.GetBillingEventId())

	// 1. Read BillingEvent + sanity-check.
	evResp, err := uc.repositories.BillingEvent.ReadBillingEvent(ctx, &billingeventpb.ReadBillingEventRequest{
		Data: &billingeventpb.BillingEvent{Id: billingEventID},
	})
	if err != nil || evResp == nil || len(evResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_not_found",
			"Billing event not found [DEFAULT]",
		))
	}
	ev := evResp.GetData()[0]

	if ev.GetSubscriptionId() != sub.GetId() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_mismatch",
			"Billing event does not belong to this subscription [DEFAULT]",
		))
	}

	switch ev.GetStatus() {
	case billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY,
		billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_DEFERRED:
		// ok
	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_not_ready",
			"Billing event is not ready to be invoiced [DEFAULT]",
		))
	}

	// 2. Idempotency — billing_event_id is the milestone key. We scan prior
	// revenues for the same subscription so the existing list-cache pattern
	// stays unified between the recurring and milestone branches.
	priorRevenues := uc.listRevenuesForSubscription(ctx, sub.GetId())
	for _, rev := range priorRevenues {
		if rev.GetStatus() == "cancelled" {
			continue
		}
		if rev.GetBillingEventId() == ev.GetId() {
			conflictID := rev.GetId()
			resp := &revenuepb.CreateRevenueWithLineItemsResponse{
				Success:              false,
				ConflictingRevenueId: &conflictID,
				Error: &commonpb.Error{
					Code: "conflicting_revenue_id",
					Message: contextutil.GetTranslatedMessageWithContext(
						ctx, uc.services.Translator,
						"revenue.errors.milestone_already_invoiced",
						"This milestone has already been invoiced [DEFAULT]",
					),
				},
			}
			return resp, errors.New(resp.GetError().GetMessage())
		}
	}

	// 3. Resolve target amount (override wins, else full event amount).
	originalEventAmount := ev.GetBillableAmount()
	target := originalEventAmount
	if req.OverrideTotalAmount != nil {
		target = req.GetOverrideTotalAmount()
	}
	if target <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.invalid_target_amount",
			"Bill amount must be greater than zero [DEFAULT]",
		))
	}
	if target > originalEventAmount {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.target_exceeds_event",
			"Bill amount cannot exceed the milestone amount [DEFAULT]",
		))
	}

	// 4. Over-billing guard — sum of all events under the same template phase
	// for the same subscription must not exceed the template-resolved total.
	if jtpID := strings.TrimSpace(ev.GetJobTemplatePhaseId()); jtpID != "" {
		templateTotal := uc.resolveTemplatePhaseAmount(ctx, jtpID, pricePlan)
		if templateTotal > 0 {
			otherSum := uc.sumBillableUnderPhase(ctx, sub.GetId(), jtpID, ev.GetId())
			if otherSum+target > templateTotal {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.Translator,
					"revenue.errors.over_billing_rejected",
					"Total billed under this milestone would exceed the template amount [DEFAULT]",
				))
			}
		}
	}

	// 5. Build line items from PPPs gated by this template phase. Operator
	// per-line overrides apply just like in the recurring branch.
	allPPPs := uc.listProductPricePlans(ctx, pricePlan.GetId())
	gatedPPPs := filterPPPsByTemplatePhase(allPPPs, ev.GetJobTemplatePhaseId())
	overridesByPPP := indexOverrides(req.GetOverrides())
	var lines []*revenuelineitempb.RevenueLineItem
	var treatments []string
	for _, ppp := range gatedPPPs {
		if ov := overridesByPPP[ppp.GetId()]; ov != nil && ov.GetRemoved() {
			continue
		}
		unitPrice := ppp.GetBillingAmount()
		quantity := 1.0
		description := describeLine(ppp)
		if ov := overridesByPPP[ppp.GetId()]; ov != nil {
			if ov.UnitPrice != nil {
				unitPrice = *ov.UnitPrice
			}
			if ov.Quantity != nil {
				quantity = *ov.Quantity
			}
			if ov.Description != nil && *ov.Description != "" {
				description = *ov.Description
			}
		}
		totalPrice := int64(float64(unitPrice) * quantity)
		pppID := ppp.GetId()
		lines = append(lines, &revenuelineitempb.RevenueLineItem{
			Description:        description,
			Quantity:           quantity,
			UnitPrice:          unitPrice,
			TotalPrice:         totalPrice,
			LineAmount:         totalPrice, // Phase 4 C2: mirrors TotalPrice for tax base calculation.
			LineItemType:       "item",
			ProductPricePlanId: &pppID,
		})
		treatments = append(treatments, treatmentOneTime)
	}

	// Empty-line guard: when no PPP is gated by this phase (or operator removed
	// all of them), fall back to a single bundle line so the milestone is still
	// represented as a billable row.
	if len(lines) == 0 {
		bundleName := pricePlan.GetName()
		if bundleName == "" {
			bundleName = "Milestone"
		}
		lines = []*revenuelineitempb.RevenueLineItem{
			{
				Description:  bundleName,
				Quantity:     1,
				UnitPrice:    target,
				TotalPrice:   target,
				LineAmount:   target, // Phase 4 C2: mirrors TotalPrice for tax base calculation.
				LineItemType: "item",
			},
		}
		treatments = []string{treatmentOneTime}
	}

	// When the operator overrides the total, scale lines proportionally so
	// they sum to the target. This is a soft compromise — full per-line
	// editing arrives via Phase D's drawer overrides; here we only need a
	// numerically-coherent invoice.
	if req.OverrideTotalAmount != nil {
		scaleLineItemsToTarget(lines, target)
	}

	previewLines := buildPreviewLines(lines, treatments)

	if req.GetDryRun() {
		return &revenuepb.CreateRevenueWithLineItemsResponse{
			Success:      true,
			PreviewLines: previewLines,
		}, nil
	}

	// 6. Atomic write — Revenue header, line items, then BillingEvent mutation.
	revenueDate := strings.TrimSpace(req.GetRevenueDate())
	if revenueDate == "" {
		revenueDate = time.Now().UTC().Format("2006-01-02")
	}

	header := uc.buildHeader(req, sub, pricePlan, priceSchedule, client, planCurrency, "", "", revenueDate, target, nil)
	// Wire milestone traceability FKs onto the header.
	if jpID := ev.GetJobPhaseId(); jpID != "" {
		jp := jpID
		header.JobPhaseId = &jp
	}
	beID := ev.GetId()
	header.BillingEventId = &beID

	createdRevenue, err := uc.persistRevenue(ctx, header)
	if err != nil {
		return nil, err
	}
	if err := uc.persistLineItems(ctx, createdRevenue.GetId(), lines); err != nil {
		return nil, err
	}

	// Phase 4 C6 — post-recognize tax compute (non-fatal).
	var milestoneWarnings []string
	if uc.services.ComputeTaxes != nil {
		wsID := sub.GetWorkspaceId()
		if computeErr := uc.services.ComputeTaxes.ExecuteForRevenue(ctx, createdRevenue.GetId(), wsID); computeErr != nil {
			milestoneWarnings = append(milestoneWarnings,
				fmt.Sprintf("tax_compute_failed: %s", computeErr.Error()))
		}
	}

	// Mutate the original event to BILLED at the actual amount.
	mutated := proto.Clone(ev).(*billingeventpb.BillingEvent)
	mutated.Status = billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED
	mutated.BillableAmount = target
	revenueID := createdRevenue.GetId()
	mutated.RevenueId = &revenueID
	billedAt := time.Now().UnixMilli()
	mutated.BilledAt = &billedAt
	if reason := strings.TrimSpace(req.GetPartialReason()); reason != "" {
		mutated.Reason = &reason
	}
	if _, err := uc.repositories.BillingEvent.UpdateBillingEvent(
		ctx, &billingeventpb.UpdateBillingEventRequest{Data: mutated},
	); err != nil {
		return nil, fmt.Errorf("update billing_event: %w", err)
	}

	// 7. Optional DEFERRED child for the unbilled remainder.
	if req.GetLeaveRemainderOpen() && target < originalEventAmount {
		remainder := originalEventAmount - target
		child := &billingeventpb.BillingEvent{
			Active:          true,
			SubscriptionId:  ev.GetSubscriptionId(),
			BillableAmount:  remainder,
			BillingCurrency: ev.GetBillingCurrency(),
			Status:          billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_DEFERRED,
			Trigger:         billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_MANUAL_LATE,
		}
		if v := ev.GetJobId(); v != "" {
			child.JobId = &v
		}
		if v := ev.GetJobPhaseId(); v != "" {
			child.JobPhaseId = &v
		}
		if v := ev.GetJobTemplatePhaseId(); v != "" {
			child.JobTemplatePhaseId = &v
		}
		if v := ev.GetProductPricePlanId(); v != "" {
			child.ProductPricePlanId = &v
		}
		parent := ev.GetId()
		child.ParentEventId = &parent
		seqLabel := nextPartialLabel(ev.GetSequenceLabel())
		child.SequenceLabel = &seqLabel
		if uc.services.IDGenerator != nil {
			child.Id = uc.services.IDGenerator.GenerateID()
		}
		now := time.Now()
		dc := now.UnixMilli()
		dcs := now.Format(time.RFC3339)
		child.DateCreated = &dc
		child.DateCreatedString = &dcs
		child.DateModified = &dc
		child.DateModifiedString = &dcs
		if _, err := uc.repositories.BillingEvent.CreateBillingEvent(
			ctx, &billingeventpb.CreateBillingEventRequest{Data: child},
		); err != nil {
			return nil, fmt.Errorf("create deferred billing_event: %w", err)
		}
	}

	return &revenuepb.CreateRevenueWithLineItemsResponse{
		Success:      true,
		Data:         []*revenuepb.Revenue{createdRevenue},
		PreviewLines: previewLines,
		Warnings:     milestoneWarnings,
	}, nil
}

// adHocPoolNotesMarker is the idempotency token written into Revenue.notes for
// AD_HOC × TOTAL_PACKAGE pool invoices. Per ad-hoc plan §4.1: idempotency keys
// on (subscription_id, "pool_initial"). We scan revenue.notes for this marker
// rather than adding a dedicated DB column — keeps the proto stable and
// matches the existing recurring-period notes-scan pattern.
const adHocPoolNotesMarker = "ad_hoc_pool_initial"

// executeAdHoc handles BILLING_KIND_AD_HOC plans for both variants:
//
//	TOTAL_PACKAGE  — prepaid pool; one Revenue per subscription, marker
//	                 "ad_hoc_pool_initial" in notes; idempotent on second call.
//	PER_OCCURRENCE — pay-per-call; one Revenue per BillingEvent; idempotent
//	                 on billing_event_id (DB partial unique index from
//	                 20260501120010 backstops).
//
// Currency + client-drift checks already happened in Execute (callsite); we
// receive client + planCurrency pre-resolved so this function only worries
// about AD_HOC-specific shape.
//
// See docs/plan/20260501-ad-hoc-subscription-billing/plan.md §4.1 + §4.2.
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeAdHoc(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	priceSchedule *priceschedulepb.PriceSchedule,
	client *clientpb.Client,
	planCurrency string,
	billingEventID string,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	basis := pricePlan.GetAmountBasis()
	switch basis {
	case priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE:
		return uc.executeAdHocPool(ctx, req, sub, pricePlan, priceSchedule, client, planCurrency)
	case priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE:
		return uc.executeAdHocPerCall(ctx, req, sub, pricePlan, priceSchedule, client, planCurrency, billingEventID)
	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.ad_hoc_invalid_basis",
			"AD_HOC plans require amount_basis = TOTAL_PACKAGE or PER_OCCURRENCE [DEFAULT]",
		))
	}
}

// executeAdHocPool — prepaid pool, single bundle invoice covering all entitled
// usages. Revenue.period_start/end are NULL (the pool is timeless); operator
// can re-call without double-creating thanks to the notes-marker idempotency.
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeAdHocPool(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	priceSchedule *priceschedulepb.PriceSchedule,
	client *clientpb.Client,
	planCurrency string,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	// Idempotency — scan prior revenues on this subscription for the
	// pool-initial marker.
	priorRevenues := uc.listRevenuesForSubscription(ctx, sub.GetId())
	for _, rev := range priorRevenues {
		if rev.GetStatus() == "cancelled" {
			continue
		}
		if strings.Contains(rev.GetNotes(), adHocPoolNotesMarker) {
			conflictID := rev.GetId()
			resp := &revenuepb.CreateRevenueWithLineItemsResponse{
				Success:              false,
				ConflictingRevenueId: &conflictID,
				Error: &commonpb.Error{
					Code: "conflicting_revenue_id",
					Message: contextutil.GetTranslatedMessageWithContext(
						ctx, uc.services.Translator,
						"revenue.errors.ad_hoc_pool_already_invoiced",
						"This pool subscription has already been invoiced [DEFAULT]",
					),
				},
			}
			return resp, errors.New(resp.GetError().GetMessage())
		}
	}

	target := pricePlan.GetBillingAmount()
	if target <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.invalid_target_amount",
			"Bill amount must be greater than zero [DEFAULT]",
		))
	}

	// Resolve entitled count (subscription override beats template — codex MAJ-1).
	entitled := pricePlan.GetEntitledOccurrences()
	if v := sub.GetEntitledOccurrencesOverride(); v > 0 {
		entitled = v
	}

	bundleName := pricePlan.GetName()
	if bundleName == "" {
		bundleName = "Pool"
	}
	if entitled > 0 {
		bundleName = fmt.Sprintf("%s — %d entitlements", bundleName, entitled)
	}

	lines := []*revenuelineitempb.RevenueLineItem{
		{
			Description:  bundleName,
			Quantity:     1,
			UnitPrice:    target,
			TotalPrice:   target,
			LineAmount:   target, // Phase 4 C2: mirrors TotalPrice for tax base calculation.
			LineItemType: "item",
		},
	}
	previewLines := buildPreviewLines(lines, []string{treatmentOneTime})

	if req.GetDryRun() {
		return &revenuepb.CreateRevenueWithLineItemsResponse{
			Success:      true,
			PreviewLines: previewLines,
		}, nil
	}

	revenueDate := strings.TrimSpace(req.GetRevenueDate())
	if revenueDate == "" {
		revenueDate = time.Now().UTC().Format("2006-01-02")
	}

	// buildHeader mixes the period suffix into the name + builds the notes
	// string — pass the marker through warnings so it lands in notes.
	header := uc.buildHeader(req, sub, pricePlan, priceSchedule, client, planCurrency,
		"", "", revenueDate, target, []string{adHocPoolNotesMarker})

	createdRevenue, err := uc.persistRevenue(ctx, header)
	if err != nil {
		return nil, err
	}
	if err := uc.persistLineItems(ctx, createdRevenue.GetId(), lines); err != nil {
		return nil, err
	}

	// Phase 4 C6 — post-recognize tax compute (non-fatal).
	var poolWarnings []string
	if uc.services.ComputeTaxes != nil {
		wsID := sub.GetWorkspaceId()
		if computeErr := uc.services.ComputeTaxes.ExecuteForRevenue(ctx, createdRevenue.GetId(), wsID); computeErr != nil {
			poolWarnings = append(poolWarnings,
				fmt.Sprintf("tax_compute_failed: %s", computeErr.Error()))
		}
	}

	return &revenuepb.CreateRevenueWithLineItemsResponse{
		Success:      true,
		Data:         []*revenuepb.Revenue{createdRevenue},
		PreviewLines: previewLines,
		Warnings:     poolWarnings,
	}, nil
}

// executeAdHocPerCall — pay-per-call, single line per delivered usage. Revenue
// is keyed on billing_event_id (partial unique index from 20260501120010
// guarantees no double-insert at the DB level — codex CRIT-3).
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeAdHocPerCall(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
	sub *subscriptionpb.Subscription,
	pricePlan *priceplanpb.PricePlan,
	priceSchedule *priceschedulepb.PriceSchedule,
	client *clientpb.Client,
	planCurrency string,
	billingEventID string,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	if billingEventID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.ad_hoc_per_call_event_required",
			"A billing event is required for per-occurrence AD_HOC plans [DEFAULT]",
		))
	}
	if uc.repositories.BillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_repository_unavailable",
			"Billing event repository is not configured [DEFAULT]",
		))
	}

	// Read + validate the BillingEvent.
	evResp, err := uc.repositories.BillingEvent.ReadBillingEvent(ctx, &billingeventpb.ReadBillingEventRequest{
		Data: &billingeventpb.BillingEvent{Id: billingEventID},
	})
	if err != nil || evResp == nil || len(evResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_not_found",
			"Billing event not found [DEFAULT]",
		))
	}
	ev := evResp.GetData()[0]
	if ev.GetSubscriptionId() != sub.GetId() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_mismatch",
			"Billing event does not belong to this subscription [DEFAULT]",
		))
	}
	if ev.GetTrigger() != billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_VISIT_COMPLETED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.ad_hoc_event_not_ready",
			"Usage is not yet completed; cannot recognize revenue [DEFAULT]",
		))
	}
	if ev.GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.billing_event_not_ready",
			"Billing event is not ready to be invoiced [DEFAULT]",
		))
	}

	// Application-side idempotency mirrors milestone billing — DB partial
	// unique index (codex CRIT-3) is the ultimate backstop.
	priorRevenues := uc.listRevenuesForSubscription(ctx, sub.GetId())
	for _, rev := range priorRevenues {
		if rev.GetStatus() == "cancelled" {
			continue
		}
		if rev.GetBillingEventId() == ev.GetId() {
			conflictID := rev.GetId()
			resp := &revenuepb.CreateRevenueWithLineItemsResponse{
				Success:              false,
				ConflictingRevenueId: &conflictID,
				Error: &commonpb.Error{
					Code: "conflicting_revenue_id",
					Message: contextutil.GetTranslatedMessageWithContext(
						ctx, uc.services.Translator,
						"revenue.errors.ad_hoc_per_call_already_invoiced",
						"This usage has already been invoiced [DEFAULT]",
					),
				},
			}
			return resp, errors.New(resp.GetError().GetMessage())
		}
	}

	target := ev.GetBillableAmount()
	if target <= 0 {
		target = pricePlan.GetBillingAmount()
	}
	if target <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.errors.invalid_target_amount",
			"Bill amount must be greater than zero [DEFAULT]",
		))
	}

	// Build the per-usage line. Description includes the event id suffix so
	// operators can correlate the line back to the BillingEvent without a
	// follow-on Job lookup. Phase D's drawer will enrich this with the
	// usage_ordinal + request_date once the centymo view layer reads the Job.
	desc := pricePlan.GetName()
	if desc == "" {
		desc = "Usage"
	}
	if evShort := ev.GetId(); len(evShort) > 0 {
		short := evShort
		if len(short) > 8 {
			short = short[len(short)-8:]
		}
		desc = fmt.Sprintf("%s — Usage (%s)", desc, short)
	}
	periodStart := ""
	periodEnd := ""

	lines := []*revenuelineitempb.RevenueLineItem{
		{
			Description:  desc,
			Quantity:     1,
			UnitPrice:    target,
			TotalPrice:   target,
			LineAmount:   target, // Phase 4 C2: mirrors TotalPrice for tax base calculation.
			LineItemType: "item",
		},
	}
	previewLines := buildPreviewLines(lines, []string{treatmentOneTime})

	if req.GetDryRun() {
		return &revenuepb.CreateRevenueWithLineItemsResponse{
			Success:      true,
			PreviewLines: previewLines,
		}, nil
	}

	revenueDate := strings.TrimSpace(req.GetRevenueDate())
	if revenueDate == "" {
		revenueDate = time.Now().UTC().Format("2006-01-02")
	}

	header := uc.buildHeader(req, sub, pricePlan, priceSchedule, client, planCurrency,
		periodStart, periodEnd, revenueDate, target, nil)
	beID := ev.GetId()
	header.BillingEventId = &beID
	if v := ev.GetJobPhaseId(); v != "" {
		header.JobPhaseId = &v
	}

	createdRevenue, err := uc.persistRevenue(ctx, header)
	if err != nil {
		return nil, err
	}
	if err := uc.persistLineItems(ctx, createdRevenue.GetId(), lines); err != nil {
		return nil, err
	}

	// Phase 4 C6 — post-recognize tax compute (non-fatal).
	var perCallWarnings []string
	if uc.services.ComputeTaxes != nil {
		wsID := sub.GetWorkspaceId()
		if computeErr := uc.services.ComputeTaxes.ExecuteForRevenue(ctx, createdRevenue.GetId(), wsID); computeErr != nil {
			perCallWarnings = append(perCallWarnings,
				fmt.Sprintf("tax_compute_failed: %s", computeErr.Error()))
		}
	}

	// Mark the BillingEvent BILLED.
	mutated := proto.Clone(ev).(*billingeventpb.BillingEvent)
	mutated.Status = billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED
	mutated.BillableAmount = target
	revenueID := createdRevenue.GetId()
	mutated.RevenueId = &revenueID
	billedAt := time.Now().UnixMilli()
	mutated.BilledAt = &billedAt
	if _, err := uc.repositories.BillingEvent.UpdateBillingEvent(
		ctx, &billingeventpb.UpdateBillingEventRequest{Data: mutated},
	); err != nil {
		return nil, fmt.Errorf("update billing_event: %w", err)
	}

	return &revenuepb.CreateRevenueWithLineItemsResponse{
		Success:      true,
		Data:         []*revenuepb.Revenue{createdRevenue},
		PreviewLines: previewLines,
		Warnings:     perCallWarnings,
	}, nil
}

// resolveTemplatePhaseAmount returns the resolved monetary cap for a template
// phase under a given PricePlan. Order of resolution:
//
//  1. Fixed billing_amount on the JobTemplatePhase.
//  2. billing_percent_bps × pricePlan.billing_amount / 10000.
//  3. Sum of ProductPricePlan amounts gated by this phase (FK match).
//
// Returns 0 when no rule matches — callers should treat 0 as "no over-billing
// cap configured" and skip the guard.
func (uc *RecognizeRevenueFromSubscriptionUseCase) resolveTemplatePhaseAmount(
	ctx context.Context, jobTemplatePhaseID string, pricePlan *priceplanpb.PricePlan,
) int64 {
	if uc.repositories.JobTemplatePhase == nil {
		return 0
	}
	resp, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, &jobtemplatephasepb.ReadJobTemplatePhaseRequest{
		Data: &jobtemplatephasepb.JobTemplatePhase{Id: jobTemplatePhaseID},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return 0
	}
	phase := resp.GetData()[0]
	if v := phase.GetBillingAmount(); v > 0 {
		return v
	}
	if pct := phase.GetBillingPercentBps(); pct > 0 {
		return (pricePlan.GetBillingAmount() * int64(pct)) / 10000
	}
	// Derive from gated PPPs.
	var sum int64
	for _, ppp := range uc.listProductPricePlans(ctx, pricePlan.GetId()) {
		if ppp.GetJobTemplatePhaseId() == jobTemplatePhaseID {
			sum += ppp.GetBillingAmount()
		}
	}
	return sum
}

// sumBillableUnderPhase returns the total committed billable amount across
// every other BillingEvent (excluding the current one) under the same
// (subscription, template_phase) combo whose status is one of the
// reservation states (READY, BILLED, DEFERRED). Used to enforce the
// over-billing guard from flow.md §11.
func (uc *RecognizeRevenueFromSubscriptionUseCase) sumBillableUnderPhase(
	ctx context.Context, subscriptionID, jobTemplatePhaseID, currentEventID string,
) int64 {
	if uc.repositories.BillingEvent == nil {
		return 0
	}
	resp, err := uc.repositories.BillingEvent.ListBySubscription(
		ctx, &billingeventpb.ListBillingEventsBySubscriptionRequest{SubscriptionId: subscriptionID},
	)
	if err != nil || resp == nil {
		return 0
	}
	var sum int64
	for _, e := range resp.GetBillingEvents() {
		if e.GetId() == currentEventID {
			continue
		}
		if e.GetJobTemplatePhaseId() != jobTemplatePhaseID {
			continue
		}
		switch e.GetStatus() {
		case billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY,
			billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED,
			billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_DEFERRED:
			sum += e.GetBillableAmount()
		}
	}
	return sum
}

// filterPPPsByTemplatePhase returns the PPPs whose job_template_phase_id
// matches the supplied id. When id is empty (event not gated by a phase),
// returns the input unchanged so all PPPs flow through.
func filterPPPsByTemplatePhase(
	ppps []*productpriceplanpb.ProductPricePlan, jobTemplatePhaseID string,
) []*productpriceplanpb.ProductPricePlan {
	if jobTemplatePhaseID == "" {
		return ppps
	}
	out := make([]*productpriceplanpb.ProductPricePlan, 0, len(ppps))
	for _, ppp := range ppps {
		if ppp.GetJobTemplatePhaseId() == jobTemplatePhaseID {
			out = append(out, ppp)
		}
	}
	return out
}

// scaleLineItemsToTarget rewrites unit_price/total_price on lines so the sum
// equals target. Last-line-fixup absorbs the rounding loss. Quantity is left
// untouched — operators see the same per-line fractional breakdown they'd
// get from a manual override.
func scaleLineItemsToTarget(lines []*revenuelineitempb.RevenueLineItem, target int64) {
	if len(lines) == 0 || target <= 0 {
		return
	}
	var current int64
	for _, l := range lines {
		current += l.GetTotalPrice()
	}
	if current == target || current <= 0 {
		// Already correct or zero-valued — write the target onto a single line
		// so callers downstream still produce a coherent header sum.
		if current != target && len(lines) == 1 {
			lines[0].UnitPrice = target
			lines[0].TotalPrice = target
		}
		return
	}
	var assigned int64
	for i, l := range lines {
		var allocated int64
		if i == len(lines)-1 {
			allocated = target - assigned
		} else {
			allocated = (l.GetTotalPrice() * target) / current
			assigned += allocated
		}
		l.TotalPrice = allocated
		if l.GetQuantity() > 0 {
			l.UnitPrice = int64(float64(allocated) / l.GetQuantity())
		} else {
			l.UnitPrice = allocated
		}
	}
}

// nextPartialLabel returns the sequence label for a child DEFERRED event
// spawned by leave_remainder_open. We deliberately keep this a string —
// operators understand "M3 partial #2" better than a structured field.
func nextPartialLabel(parentLabel string) string {
	if strings.TrimSpace(parentLabel) == "" {
		return "partial #2"
	}
	return parentLabel + " (continued)"
}
