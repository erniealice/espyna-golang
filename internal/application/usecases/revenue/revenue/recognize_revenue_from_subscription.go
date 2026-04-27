package revenue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

const (
	entitySubscription = "subscription"

	// Treatment tokens used for the preview badge — kept lowercase so the
	// drawer's lyngua key map is straightforward.
	treatmentRecurring   = "recurring"
	treatmentFirstCycle  = "first_cycle"
	treatmentUsageBased  = "usage_based"
	treatmentOneTime     = "one_time"
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
}

// RecognizeRevenueFromSubscriptionServices groups all business service
// dependencies. Mirrors CreateRevenueServices.
type RecognizeRevenueFromSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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

// Execute orchestrates the revenue recognition flow. The shape of the request
// matches the CreateRevenueWithLineItems RPC; when dry_run is set the use case
// returns a preview without writing.
func (uc *RecognizeRevenueFromSubscriptionUseCase) Execute(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionCreate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySubscription, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.validation.request_required",
			"Request is required [DEFAULT]",
		))
	}

	subscriptionID := req.GetSubscriptionId()
	if subscriptionID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.validation.subscription_id_required",
			"subscription_id is required [DEFAULT]",
		))
	}

	if uc.services.TransactionService != nil &&
		uc.services.TransactionService.SupportsTransactions() &&
		!req.GetDryRun() {
		var result *revenuepb.CreateRevenueWithLineItemsResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore is the transaction-agnostic body. It runs all reads, builds the
// preview, and (unless dry_run) writes the Revenue + lines.
func (uc *RecognizeRevenueFromSubscriptionUseCase) executeCore(
	ctx context.Context,
	req *revenuepb.CreateRevenueWithLineItemsRequest,
) (*revenuepb.CreateRevenueWithLineItemsResponse, error) {
	subscriptionID := req.GetSubscriptionId()

	// 1. Resolve subscription
	sub, err := uc.readSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if !sub.GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.subscription_inactive",
			"Subscription is inactive [DEFAULT]",
		))
	}

	// 2. Resolve price plan
	pricePlanID := sub.GetPricePlanId()
	if pricePlanID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"subscription.errors.price_plan_required",
			"Subscription has no price plan [DEFAULT]",
		))
	}
	pricePlan, err := uc.readPricePlan(ctx, pricePlanID)
	if err != nil {
		return nil, err
	}

	// 3. Hydrate context (client, payment term, price schedule)
	client := uc.readClient(ctx, sub.GetClientId())
	priceSchedule := uc.readPriceSchedule(ctx, pricePlan.GetPriceScheduleId())

	// 4. Currency assertion (hard block per plan §11.4)
	planCurrency := pricePlan.GetBillingCurrency()
	clientCurrency := ""
	if client != nil {
		clientCurrency = client.GetBillingCurrency()
	}
	if clientCurrency != "" && planCurrency != "" && clientCurrency != planCurrency {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
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
			ctx, uc.services.TranslationService,
			"revenue.errors.subscriptionPlanClientDrift",
			"Subscription and price plan belong to different clients — recognition blocked. [DEFAULT]",
		))
	}

	// 5. List ProductPricePlans for this PricePlan
	ppps := uc.listProductPricePlans(ctx, pricePlanID)

	// 6. Idempotency + first-cycle detection require existing revenue rows
	priorRevenues := uc.listRevenuesForSubscription(ctx, subscriptionID)
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
				Code:    "period_already_invoiced",
				Message: contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.TranslationService,
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

	// 8. Build line items per plan §3.3
	var warnings []string
	overridesByPPP := indexOverrides(req.GetOverrides())
	lines, treatments, lineWarnings := buildLineItems(pricePlan, ppps, overridesByPPP, priorLinesByPPP)
	warnings = append(warnings, lineWarnings...)

	// 9. Empty-PPP fallback for TOTAL_PACKAGE plans (edge case 9 in plan)
	if len(lines) == 0 && pricePlan.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		lines, treatments = buildBundleLine(pricePlan)
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
				ctx, uc.services.TranslationService,
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

	createdRevenue, err := uc.persistRevenue(ctx, header)
	if err != nil {
		return nil, err
	}

	if err := uc.persistLineItems(ctx, createdRevenue.GetId(), lines); err != nil {
		return nil, err
	}

	return &revenuepb.CreateRevenueWithLineItemsResponse{
		Success:      true,
		Data:         []*revenuepb.Revenue{createdRevenue},
		PreviewLines: previewLines,
		Warnings:     warnings,
	}, nil
}

// readSubscription wraps the subscription RPC for clarity.
func (uc *RecognizeRevenueFromSubscriptionUseCase) readSubscription(
	ctx context.Context, id string,
) (*subscriptionpb.Subscription, error) {
	if uc.repositories.Subscription == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
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
			ctx, uc.services.TranslationService,
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
			ctx, uc.services.TranslationService,
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
			ctx, uc.services.TranslationService,
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
// Filtering happens in Go because the proto's ListProductPricePlansRequest
// does not yet take a price_plan_id (matches the existing autoPopulateLineItems
// pattern in centymo).
func (uc *RecognizeRevenueFromSubscriptionUseCase) listProductPricePlans(
	ctx context.Context, pricePlanID string,
) []*productpriceplanpb.ProductPricePlan {
	if uc.repositories.ProductPricePlan == nil {
		return nil
	}
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
			Description:        description,
			Quantity:           quantity,
			UnitPrice:          unitPrice,
			TotalPrice:         totalPrice,
			LineItemType:       "item",
			ProductPricePlanId: stringPtrLocal(pppID),
		}
		_ = currency // currency lives on the Revenue header — RevenueLineItem proto has no currency field
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
	if header.Id == "" && uc.services.IDService != nil {
		header.Id = uc.services.IDService.GenerateID()
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
			ctx, uc.services.TranslationService,
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
			ctx, uc.services.TranslationService,
			"revenue_line_item.errors.repository_unavailable",
			"Revenue line item repository is not configured [DEFAULT]",
		))
	}
	for _, l := range lines {
		if l.Id == "" && uc.services.IDService != nil {
			l.Id = uc.services.IDService.GenerateID()
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
