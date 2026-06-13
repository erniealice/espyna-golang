package revenue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	serviceamortization "github.com/erniealice/espyna-golang/internal/application/usecases/service/amortization"
	treasurycollection "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
	amortizationpb "github.com/erniealice/esqyma/pkg/schema/v1/service/amortization"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// RevenueRunScope describes the filter for a candidate list or generate call.
// All fields are optional; an empty scope returns all active cyclic subscriptions
// in the workspace.
type RevenueRunScope struct {
	WorkspaceID    string
	ClientID       string
	SubscriptionID string
	AsOfDate       string // YYYY-MM-DD; defaults to today when empty
	Cursor         string
	Limit          int32
	// IncludeAdvanceCollections opts-in to advance-Collection candidate
	// emission (Plan B Phase 5a). When false (default), only subscription-cycle
	// candidates are emitted — preserves pre-5a behavior for callers that
	// haven't opted in.
	IncludeAdvanceCollections bool
}

// periodWindow holds the computed start/end for one billing period.
type periodWindow struct {
	Start string // YYYY-MM-DD
	End   string // YYYY-MM-DD
}

// ListRevenueRunCandidatesRepositories groups all repository dependencies.
type ListRevenueRunCandidatesRepositories struct {
	Revenue      revenuepb.RevenueDomainServiceServer
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
	// Workspace repo — used to resolve workspace.timezone for billing-cycle math.
	// Optional; when nil, period enumeration falls back to UTC truncation
	// (preserves pre-timezone-aware behavior).
	Workspace workspacepb.WorkspaceDomainServiceServer
	// TreasuryCollection — used to enumerate active TIME_BASED advance
	// Collections when IncludeAdvanceCollections is true (Plan B Phase 5a).
	// Optional; when nil, advance-Collection candidates are silently skipped
	// even when the request opts in.
	TreasuryCollection collectionpb.CollectionDomainServiceServer
}

// ListRevenueRunCandidatesServices groups all business service dependencies.
type ListRevenueRunCandidatesServices struct {
	Authorizer   ports.Authorizer
	Translator   ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// Amortization is the service-driven amortization schedule wrapper.
	// Used by buildAdvanceCandidate to compute next-due tranches with
	// proto-typed IO. Wired by the composition root via
	// serviceamortization.From(serviceUseCases).
	Amortization *serviceamortization.ComputeNextDueTrancheUseCase
}

// ListRevenueRunCandidatesUseCase enumerates pending billing periods for the
// given scope and performs a dry-run through RecognizeRevenueFromSubscription
// to validate each period before surfacing it to the operator.
type ListRevenueRunCandidatesUseCase struct {
	repositories     ListRevenueRunCandidatesRepositories
	services         ListRevenueRunCandidatesServices
	recognizeUseCase *RecognizeRevenueFromSubscriptionUseCase
}

// NewListRevenueRunCandidatesUseCase wires the use case.
func NewListRevenueRunCandidatesUseCase(
	repositories ListRevenueRunCandidatesRepositories,
	services ListRevenueRunCandidatesServices,
	recognizeUseCase *RecognizeRevenueFromSubscriptionUseCase,
) *ListRevenueRunCandidatesUseCase {
	return &ListRevenueRunCandidatesUseCase{
		repositories:     repositories,
		services:         services,
		recognizeUseCase: recognizeUseCase,
	}
}

// Execute returns the list of un-invoiced period candidates for the scope.
// When req.Limit == 0 the full result set is returned (no cursor used).
//
// Adaptations previously performed by the consumer wrapper now live inside
// Execute: nil-slice normalization (Data is always a non-nil slice) and the
// context-bound workspace-id fallback.
func (uc *ListRevenueRunCandidatesUseCase) Execute(
	ctx context.Context,
	req *revenuerunpb.ListRevenueRunCandidatesRequest,
) (*revenuerunpb.ListRevenueRunCandidatesResponse, error) {
	// 1. Auth checks
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

	// Translate the proto request to the internal Go-struct scope used by
	// helper methods. Cursor/Limit are read directly off req below.
	protoScope := req.GetScope()
	scope := RevenueRunScope{
		WorkspaceID:               protoScope.GetWorkspaceId(),
		ClientID:                  protoScope.GetClientId(),
		SubscriptionID:            protoScope.GetSubscriptionId(),
		AsOfDate:                  protoScope.GetAsOfDate(),
		IncludeAdvanceCollections: req.GetIncludeAdvanceCollections(),
	}
	limit := req.GetLimit()

	// Fall back to context-bound workspace ID when the caller didn't set one.
	// Why: view-layer scopes built in centymo/entydad drawers don't always
	// populate WorkspaceID (it's not on the request path). Without this, the
	// timezone resolution below silently falls back to UTC and period
	// enumeration projects Manila-typed start dates onto the wrong calendar day.
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		scope.WorkspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}

	// 2. Resolve workspace timezone (source of truth for "what calendar day").
	// Falls back to UTC when workspace lookup is unavailable or empty — keeps
	// pre-timezone-aware behavior intact for callers that haven't been migrated.
	loc := resolveWorkspaceLocation(ctx, uc.repositories.Workspace, scope.WorkspaceID)

	// 3. Resolve AsOfDate — default to today (in the workspace's tz)
	asOfDate := strings.TrimSpace(scope.AsOfDate)
	if asOfDate == "" {
		asOfDate = time.Now().In(loc).Format("2006-01-02")
	}
	asOfTime, err := time.ParseInLocation("2006-01-02", asOfDate, loc)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"revenue.validation.invalid_as_of_date",
			"as_of_date must be YYYY-MM-DD [DEFAULT]",
		))
	}

	// 4. List active subscriptions filtered by scope
	subs, err := uc.listSubscriptions(ctx, scope)
	if err != nil {
		return nil, err
	}

	// 4. Build existing period marker set for all relevant subscriptions
	existingMarkers := uc.buildExistingMarkerSet(ctx, subs)

	// 4c. List active TIME_BASED advance Collections in scope.
	// Used both for emitting ADVANCE_COLLECTION candidates AND for suppressing
	// subscription cycles that overlap an active advance for the same client.
	// Always loaded when the request opts in OR the workspace has any advance —
	// suppression must run even if the client is not directly using the new flag,
	// to keep Decision A (advance overrides cycle) honored.
	advances := uc.listActiveTimeBasedAdvances(ctx, scope)

	// 5. Enumerate periods per subscription. Initialised non-nil so the
	// response's Data field is never nil — consumers do not need to defend.
	candidates := []*revenuerunpb.RevenueRunCandidate{}
	for _, sub := range subs {
		// Resolve price plan for this subscription
		plan, planErr := uc.readPricePlan(ctx, sub.GetPricePlanId())
		if planErr != nil || plan == nil {
			continue
		}

		// Skip MILESTONE and AD_HOC — separate billing flows
		if !isCyclicForRunCandidates(plan) {
			continue
		}

		// Enumerate all un-invoiced periods
		windows := enumeratePeriods(sub, plan, asOfTime, loc)
		subMarkers := existingMarkers[sub.GetId()]

		for _, w := range windows {
			marker := buildPeriodMarker(w.Start, w.End)
			if subMarkers[marker] {
				continue // already invoiced
			}

			// Dry-run the recognizer to get amount, line count, and blocker flags
			candidate := uc.buildCandidate(ctx, sub, plan, w, marker)

			// Decision A — suppress subscription-cycle rows that overlap an
			// active TIME_BASED advance Collection for the same client.
			if suppressorID := findSuppressingAdvance(advances, sub.GetClientId(), w.Start, w.End); suppressorID != "" {
				candidate.Eligible = false
				candidate.BlockerReason = "suppressed_by_advance_collection"
				candidate.SuppressingAdvanceCollectionId = &suppressorID
			}
			// Tag origin for the view layer (UNSPECIFIED is treated as
			// SUBSCRIPTION_CYCLE downstream; this makes intent explicit).
			candidate.SourceKind = revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_SUBSCRIPTION_CYCLE
			candidates = append(candidates, candidate)
		}
	}

	// 5b. Emit advance-Collection candidates when the request opted in.
	// Each active TIME_BASED advance gets at most ONE candidate per call: the
	// next-due tranche per amortize_schedule. The drawer renders these as a
	// second source-kind group; idempotency-skipped tranches surface as
	// eligible=false with a "already_recognized" blocker.
	if scope.IncludeAdvanceCollections {
		for _, adv := range advances {
			cand := uc.buildAdvanceCandidate(ctx, adv, asOfDate)
			if cand != nil {
				candidates = append(candidates, cand)
			}
		}
	}

	// 6. Cursor pagination (used by Surface B)
	if limit > 0 && len(candidates) > int(limit) {
		nextCursor := fmt.Sprintf("%d", limit)
		return &revenuerunpb.ListRevenueRunCandidatesResponse{
			Data:       candidates[:limit],
			Success:    true,
			NextCursor: &nextCursor,
		}, nil
	}

	return &revenuerunpb.ListRevenueRunCandidatesResponse{
		Data:    candidates,
		Success: true,
	}, nil
}

// buildCandidate performs a dry-run and assembles a proto RevenueRunCandidate.
func (uc *ListRevenueRunCandidatesUseCase) buildCandidate(
	ctx context.Context,
	sub *subscriptionpb.Subscription,
	plan *priceplanpb.PricePlan,
	w periodWindow,
	marker string,
) *revenuerunpb.RevenueRunCandidate {
	candidate := &revenuerunpb.RevenueRunCandidate{
		SubscriptionId:    sub.GetId(),
		SubscriptionName:  sub.GetName(),
		ClientId:          sub.GetClientId(),
		PlanName:          plan.GetName(),
		BillingCycleLabel: billingCycleLabel(plan),
		Currency:          plan.GetBillingCurrency(),
		PeriodStart:       w.Start,
		PeriodEnd:         w.End,
		PeriodLabel:       fmt.Sprintf("%s – %s", w.Start, w.End),
		PeriodMarker:      marker,
	}

	if uc.recognizeUseCase == nil {
		candidate.Eligible = false
		candidate.BlockerReason = "recognizer_unavailable"
		return candidate
	}

	req := buildRecognizeRequest(sub.GetId(), w.Start, w.End, true /* DryRun */)
	resp, execErr := uc.recognizeUseCase.Execute(ctx, req)
	if execErr != nil {
		candidate.Eligible = false
		candidate.BlockerReason = extractBlockerReason(execErr)
		return candidate
	}
	if resp == nil || !resp.GetSuccess() {
		candidate.Eligible = false
		if resp != nil && resp.GetError() != nil {
			candidate.BlockerReason = resp.GetError().GetCode()
		}
		return candidate
	}

	// Success path — tally from preview lines
	var totalAmount int64
	for _, l := range resp.GetPreviewLines() {
		totalAmount += l.GetTotalPrice()
	}
	candidate.Amount = totalAmount
	candidate.LineItemCount = int32(len(resp.GetPreviewLines()))
	candidate.Eligible = true
	return candidate
}

// listSubscriptions fetches active subscriptions filtered by the scope.
func (uc *ListRevenueRunCandidatesUseCase) listSubscriptions(
	ctx context.Context,
	scope RevenueRunScope,
) ([]*subscriptionpb.Subscription, error) {
	if uc.repositories.Subscription == nil {
		return nil, nil
	}

	filters := []*commonpb.TypedFilter{
		{
			Field: "active",
			FilterType: &commonpb.TypedFilter_BooleanFilter{
				BooleanFilter: &commonpb.BooleanFilter{Value: true},
			},
		},
	}

	if scope.ClientID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "client_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.ClientID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}
	if scope.SubscriptionID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.SubscriptionID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}

	resp, err := uc.repositories.Subscription.ListSubscriptions(ctx, &subscriptionpb.ListSubscriptionsRequest{
		Filters: &commonpb.FilterRequest{Filters: filters},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.GetData(), nil
}

// readPricePlan fetches a single price plan by ID.
func (uc *ListRevenueRunCandidatesUseCase) readPricePlan(
	ctx context.Context,
	id string,
) (*priceplanpb.PricePlan, error) {
	if id == "" || uc.repositories.PricePlan == nil {
		return nil, nil
	}
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// buildExistingMarkerSet loads existing period markers for all subscriptions
// so we can skip already-invoiced periods without a per-sub DB round-trip per
// period. Returns an empty map on error.
func (uc *ListRevenueRunCandidatesUseCase) buildExistingMarkerSet(
	ctx context.Context,
	subs []*subscriptionpb.Subscription,
) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	if uc.repositories.Revenue == nil {
		return result
	}
	for _, sub := range subs {
		markers := make(map[string]bool)
		existing := uc.listRevenuesForSub(ctx, sub.GetId())
		for _, r := range existing {
			// Extract the period marker from the notes field. The marker is
			// written as the first line of notes by buildNotes() in the
			// recognizer and has the form "Period: YYYY-MM-DD → YYYY-MM-DD".
			if n := r.GetNotes(); strings.HasPrefix(n, "Period:") {
				end := strings.Index(n, "\n")
				if end == -1 {
					end = len(n)
				}
				markers[strings.TrimSpace(n[:end])] = true
			}
		}
		result[sub.GetId()] = markers
	}
	return result
}

// listRevenuesForSub fetches revenues for a subscription. Returns nil on error.
func (uc *ListRevenueRunCandidatesUseCase) listRevenuesForSub(
	ctx context.Context,
	subscriptionID string,
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

// enumeratePeriods computes all billing periods from sub.date_time_start up to
// (and including) asOfDate for the given price plan. Returns an empty slice
// when no periods are due.
//
// All date math runs in the workspace's timezone (loc). Truncation to
// "calendar day" projects the stored UTC instant into loc first, then keeps
// year/month/day, ignoring time-of-day. This is the only way to recover the
// user-intended date when proto Timestamp has discarded the original offset.
//
// Algorithm:
//  1. Determine cycle length from plan.billing_cycle_value/unit (preferred).
//     Fall back to plan.duration_value/unit (deprecated).
//  2. Walk from sub.date_time_start in cycle-length steps.
//  3. Stop when the period start exceeds asOfDate.
//  4. Cap period end at asOfDate for open-ended subs where the current cycle
//     straddles the as-of boundary.
func enumeratePeriods(
	sub *subscriptionpb.Subscription,
	plan *priceplanpb.PricePlan,
	asOfDate time.Time,
	loc *time.Location,
) []periodWindow {
	if loc == nil {
		loc = time.UTC
	}
	startTS := sub.GetDateTimeStart()
	if startTS == nil {
		return nil
	}
	subStart := truncateToDate(startTS.AsTime(), loc)

	cycleValue, cycleUnit := resolveCycleParams(plan)
	if cycleValue <= 0 || cycleUnit == "" {
		return nil
	}

	// Optional hard end — nil means open-ended
	var subEndDate *time.Time
	if endTS := sub.GetDateTimeEnd(); endTS != nil {
		t := truncateToDate(endTS.AsTime(), loc)
		subEndDate = &t
	}

	var windows []periodWindow
	periodStart := subStart

	for {
		if periodStart.After(asOfDate) {
			break
		}
		// Stop if sub has ended before this period even starts
		if subEndDate != nil && periodStart.After(*subEndDate) {
			break
		}

		// Compute next cycle boundary (exclusive end) then back off by one day
		// to get the inclusive end of this period.
		nextStart := addCycle(periodStart, cycleValue, cycleUnit)
		periodEnd := nextStart.AddDate(0, 0, -1)

		// Cap against sub hard end date
		if subEndDate != nil && periodEnd.After(*subEndDate) {
			periodEnd = *subEndDate
		}
		// For the last (current) cycle, cap at asOfDate so we don't extend
		// into the future.
		if periodEnd.After(asOfDate) {
			periodEnd = asOfDate
		}

		windows = append(windows, periodWindow{
			Start: periodStart.Format("2006-01-02"),
			End:   periodEnd.Format("2006-01-02"),
		})

		// Advance: next period starts at the next cycle boundary.
		periodStart = nextStart
	}

	return windows
}

// truncateToDate projects a UTC instant into loc and returns the calendar-day
// at midnight in loc. Replaces the old `t.UTC().Truncate(24*time.Hour)`
// pattern, which produced the wrong calendar day for any non-UTC workspace
// (the original input zone is lost when proto Timestamp serialises).
func truncateToDate(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// resolveWorkspaceLocation reads the workspace's IANA timezone and returns
// the *time.Location for it. Falls back to time.UTC when the repo is unwired,
// the workspace is missing, the timezone field is empty, or the IANA name
// fails to load. The fallback preserves pre-timezone-aware behavior — callers
// that haven't been migrated continue to see UTC-truncated dates.
//
// Package-level free function so siblings in this package (e.g.
// GenerateRevenueRunUseCase) can share the same TZ resolution without
// duplicating the logic.
func resolveWorkspaceLocation(ctx context.Context, repo workspacepb.WorkspaceDomainServiceServer, workspaceID string) *time.Location {
	if repo == nil || strings.TrimSpace(workspaceID) == "" {
		return time.UTC
	}
	resp, err := repo.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: workspaceID},
	})
	if err != nil || resp == nil || !resp.GetSuccess() || len(resp.GetData()) == 0 {
		return time.UTC
	}
	tz := strings.TrimSpace(resp.GetData()[0].GetTimezone())
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// resolveCycleParams returns the billing cycle value and unit from the plan.
// billing_cycle_value/unit is preferred; duration_value/unit is the deprecated fallback.
func resolveCycleParams(plan *priceplanpb.PricePlan) (int, string) {
	if plan == nil {
		return 0, ""
	}
	if v := plan.GetBillingCycleValue(); v > 0 {
		u := strings.ToLower(strings.TrimSpace(plan.GetBillingCycleUnit()))
		if u != "" {
			return int(v), u
		}
	}
	// Deprecated fallback
	if v := plan.GetDurationValue(); v > 0 {
		u := strings.ToLower(strings.TrimSpace(plan.GetDurationUnit()))
		if u != "" {
			return int(v), u
		}
	}
	return 0, ""
}

// addCycle advances t by value × unit.
func addCycle(t time.Time, value int, unit string) time.Time {
	switch unit {
	case "day":
		return t.AddDate(0, 0, value)
	case "week":
		return t.AddDate(0, 0, value*7)
	case "month":
		return t.AddDate(0, value, 0)
	case "year":
		return t.AddDate(value, 0, 0)
	default:
		return t.AddDate(0, value, 0) // default to months
	}
}

// isCyclicForRunCandidates returns true when the plan participates in the
// revenue-run period-enumeration flow. MILESTONE and AD_HOC are excluded.
func isCyclicForRunCandidates(plan *priceplanpb.PricePlan) bool {
	if plan == nil {
		return false
	}
	switch plan.GetBillingKind() {
	case priceplanpb.BillingKind_BILLING_KIND_RECURRING:
		return true
	case priceplanpb.BillingKind_BILLING_KIND_CONTRACT:
		return plan.GetBillingCycleValue() > 0 || plan.GetDurationValue() > 0
	default:
		return false
	}
}

// billingCycleLabel builds a human-readable cycle label, e.g. "1 month".
func billingCycleLabel(plan *priceplanpb.PricePlan) string {
	v, u := resolveCycleParams(plan)
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%d %s", v, u)
}

// extractBlockerReason converts a use-case error to a short code for the UI.
func extractBlockerReason(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, code := range []string{
		"currency_mismatch",
		"period_already_invoiced",
		"subscription_inactive",
		"price_plan_required",
		"no_lines_to_invoice",
	} {
		if strings.Contains(msg, code) {
			return code
		}
	}
	return "recognition_error"
}

// listActiveTimeBasedAdvances returns active TIME_BASED advance Collections
// in the workspace/client scope. Returns nil on adapter unavailable.
//
// Filters applied (postgres):
//   - active=true
//   - advance_kind=TIME_BASED (numeric enum value)
//   - advance_status=ACTIVE   (numeric enum value)
//   - workspace_id           (when scope.WorkspaceID is set)
//   - client_id              (when scope.ClientID is set)
func (uc *ListRevenueRunCandidatesUseCase) listActiveTimeBasedAdvances(
	ctx context.Context,
	scope RevenueRunScope,
) []*collectionpb.Collection {
	if uc.repositories.TreasuryCollection == nil {
		return nil
	}
	filters := []*commonpb.TypedFilter{
		{
			Field: "active",
			FilterType: &commonpb.TypedFilter_BooleanFilter{
				BooleanFilter: &commonpb.BooleanFilter{Value: true},
			},
		},
		{
			Field: "advance_kind",
			FilterType: &commonpb.TypedFilter_NumberFilter{
				NumberFilter: &commonpb.NumberFilter{
					Value:    float64(advancekindpb.AdvanceKind_ADVANCE_KIND_TIME_BASED),
					Operator: commonpb.NumberOperator_NUMBER_EQUALS,
				},
			},
		},
		{
			Field: "advance_status",
			FilterType: &commonpb.TypedFilter_NumberFilter{
				NumberFilter: &commonpb.NumberFilter{
					Value:    float64(advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE),
					Operator: commonpb.NumberOperator_NUMBER_EQUALS,
				},
			},
		},
	}
	if scope.WorkspaceID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "workspace_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.WorkspaceID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}
	if scope.ClientID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "client_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.ClientID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}
	resp, err := uc.repositories.TreasuryCollection.ListCollections(ctx, &collectionpb.ListCollectionsRequest{
		Filters: &commonpb.FilterRequest{Filters: filters},
	})
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetData()
}

// findSuppressingAdvance returns the advance Collection ID whose schedule
// overlaps the given (clientID, periodStart, periodEnd) window, or "" when
// no overlap. Implements Decision A — advance Collection trumps subscription
// cycle when both cover the same period for the same client.
func findSuppressingAdvance(
	advances []*collectionpb.Collection,
	clientID, periodStart, periodEnd string,
) string {
	if clientID == "" || periodStart == "" || periodEnd == "" {
		return ""
	}
	for _, adv := range advances {
		if adv.GetClientId() != clientID {
			continue
		}
		if !dateRangesOverlap(adv.GetAdvanceStartDate(), adv.GetAdvanceEndDate(), periodStart, periodEnd) {
			continue
		}
		return adv.GetId()
	}
	return ""
}

// dateRangesOverlap returns true when [aStart, aEnd] overlaps [bStart, bEnd].
// Empty advance_end_date is treated as open-ended (overlap-ever-after).
func dateRangesOverlap(aStart, aEnd, bStart, bEnd string) bool {
	if aStart == "" {
		return false
	}
	if aEnd != "" && aEnd < bStart {
		return false
	}
	if aStart > bEnd {
		return false
	}
	return true
}

// buildAdvanceCandidate computes the next-due tranche for one advance
// Collection and packages it as a RevenueRunCandidate row.
//
// Eligibility:
//   - ok=false (no tranche due as of date) → returns nil so the row is omitted
//   - tranche already recognized via existing Revenue → eligible=false +
//     "already_recognized" blocker (visible in the drawer as a greyed row)
//   - else eligible=true
func (uc *ListRevenueRunCandidatesUseCase) buildAdvanceCandidate(
	ctx context.Context,
	adv *collectionpb.Collection,
	asOfDate string,
) *revenuerunpb.RevenueRunCandidate {
	trancheResp, err := uc.services.Amortization.Execute(ctx, &amortizationpb.ComputeNextDueTrancheRequest{
		StartDate:       adv.GetAdvanceStartDate(),
		EndDate:         adv.GetAdvanceEndDate(),
		PeriodCount:     int32(adv.GetAdvancePeriodCount()),
		PeriodUnit:      adv.GetAdvancePeriodUnit(),
		TotalAmount:     adv.GetAdvanceTotalAmount(),
		ProrationPolicy: treasurycollection.ProtoProrationToHelper(adv.GetAdvanceProrationPolicy()),
		AsOfDate:        asOfDate,
	})
	if err != nil || !trancheResp.GetFound() {
		return nil
	}
	tranche := trancheResp.GetTranche()

	advanceID := adv.GetId()
	marker := treasurycollection.BuildAdvancePeriodMarker(tranche.GetPeriodStart(), tranche.GetPeriodEnd())
	candidate := &revenuerunpb.RevenueRunCandidate{
		ClientId:            adv.GetClientId(),
		Currency:            adv.GetCurrency(),
		PeriodStart:         tranche.GetPeriodStart(),
		PeriodEnd:           tranche.GetPeriodEnd(),
		PeriodLabel:         fmt.Sprintf("%s – %s", tranche.GetPeriodStart(), tranche.GetPeriodEnd()),
		PeriodMarker:        marker,
		Amount:              tranche.GetAmount(),
		LineItemCount:       1,
		Eligible:            true,
		SourceKind:          revenuerunpb.RevenueRunSourceKind_REVENUE_RUN_SOURCE_KIND_ADVANCE_COLLECTION,
		AdvanceCollectionId: &advanceID,
	}

	// Idempotency check — surface "already_recognized" so operators see why
	// a tranche row is greyed out.
	if uc.repositories.Revenue != nil {
		if found := advanceTrancheAlreadyRecognized(ctx, uc.repositories.Revenue, advanceID, marker); found {
			candidate.Eligible = false
			candidate.BlockerReason = "already_recognized"
		}
	}
	return candidate
}

// advanceTrancheAlreadyRecognized scans existing Revenues bound to the advance
// Collection for one whose notes-first-line marker matches the computed
// tranche. Mirrors the idempotency check inside AmortizeAdvanceCollection so
// the drawer renders the same "already done" state the actual run would.
func advanceTrancheAlreadyRecognized(
	ctx context.Context,
	repo revenuepb.RevenueDomainServiceServer,
	advanceID, wantMarker string,
) bool {
	resp, err := repo.ListRevenues(ctx, &revenuepb.ListRevenuesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "advance_collection_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    advanceID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil {
		return false
	}
	for _, r := range resp.GetData() {
		n := r.GetNotes()
		if n == "" {
			continue
		}
		first := n
		if idx := strings.Index(n, "\n"); idx >= 0 {
			first = n[:idx]
		}
		if strings.TrimSpace(first) == wantMarker {
			return true
		}
	}
	return false
}

// buildRecognizeRequest builds a CreateRevenueWithLineItemsRequest for the
// given subscription and period, with DryRun set as specified.
func buildRecognizeRequest(
	subscriptionID, periodStart, periodEnd string,
	dryRun bool,
) *revenuepb.CreateRevenueWithLineItemsRequest {
	return &revenuepb.CreateRevenueWithLineItemsRequest{
		SubscriptionId: strPtr(subscriptionID),
		PeriodStart:    strPtr(periodStart),
		PeriodEnd:      strPtr(periodEnd),
		DryRun:         boolPtr(dryRun),
	}
}

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
