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
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
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
}

// RevenueRunCandidate represents one billed period for one subscription that
// has not yet been invoiced (or that has a blocker preventing invoicing).
type RevenueRunCandidate struct {
	SubscriptionID    string
	SubscriptionName  string
	ClientID          string
	ClientName        string
	PlanName          string
	BillingCycleLabel string
	Currency          string
	PeriodStart       string // YYYY-MM-DD
	PeriodEnd         string // YYYY-MM-DD
	PeriodLabel       string
	PeriodMarker      string
	Amount            int64
	LineItemCount     int
	Eligible          bool
	BlockerReason     string
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
}

// ListRevenueRunCandidatesServices groups all business service dependencies.
type ListRevenueRunCandidatesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
// When scope.Limit == 0 the full result set is returned (no cursor used).
func (uc *ListRevenueRunCandidatesUseCase) Execute(
	ctx context.Context,
	scope RevenueRunScope,
) ([]RevenueRunCandidate, string, error) {
	// 1. Auth checks
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionCreate); err != nil {
		return nil, "", err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySubscription, ports.ActionRead); err != nil {
		return nil, "", err
	}

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
	loc := uc.resolveWorkspaceLocation(ctx, scope.WorkspaceID)

	// 3. Resolve AsOfDate — default to today (in the workspace's tz)
	asOfDate := strings.TrimSpace(scope.AsOfDate)
	if asOfDate == "" {
		asOfDate = time.Now().In(loc).Format("2006-01-02")
	}
	asOfTime, err := time.ParseInLocation("2006-01-02", asOfDate, loc)
	if err != nil {
		return nil, "", errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"revenue.validation.invalid_as_of_date",
			"as_of_date must be YYYY-MM-DD [DEFAULT]",
		))
	}

	// 4. List active subscriptions filtered by scope
	subs, err := uc.listSubscriptions(ctx, scope)
	if err != nil {
		return nil, "", err
	}

	// 4. Build existing period marker set for all relevant subscriptions
	existingMarkers := uc.buildExistingMarkerSet(ctx, subs)

	// 5. Enumerate periods per subscription
	var candidates []RevenueRunCandidate
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
			candidates = append(candidates, candidate)
		}
	}

	// 6. Cursor pagination (used by Surface B)
	if scope.Limit > 0 && len(candidates) > int(scope.Limit) {
		nextCursor := fmt.Sprintf("%d", scope.Limit)
		return candidates[:scope.Limit], nextCursor, nil
	}

	return candidates, "", nil
}

// buildCandidate performs a dry-run and assembles a RevenueRunCandidate.
func (uc *ListRevenueRunCandidatesUseCase) buildCandidate(
	ctx context.Context,
	sub *subscriptionpb.Subscription,
	plan *priceplanpb.PricePlan,
	w periodWindow,
	marker string,
) RevenueRunCandidate {
	candidate := RevenueRunCandidate{
		SubscriptionID:    sub.GetId(),
		SubscriptionName:  sub.GetName(),
		ClientID:          sub.GetClientId(),
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
	candidate.LineItemCount = len(resp.GetPreviewLines())
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
func (uc *ListRevenueRunCandidatesUseCase) resolveWorkspaceLocation(ctx context.Context, workspaceID string) *time.Location {
	if uc.repositories.Workspace == nil || strings.TrimSpace(workspaceID) == "" {
		return time.UTC
	}
	resp, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
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
