// Package expenserecognitionrun is Plan A's buying-side mirror of
// revenue.ListRevenueRunCandidates + GenerateRevenueRun, with Plan B's
// AdvanceDisbursement source kind composed in.
//
// See docs/plan/20260517-expense-run/plan.md §"Phase 2" and the hard rules
// (idempotency-check-first + compose, don't duplicate).
package expenserecognitionrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	amortizeschedule "github.com/erniealice/espyna-golang/internal/application/shared/amortize_schedule"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

const (
	entityExpenseRecognition   = "expense_recognition"
	entitySupplierSubscription = "supplier_subscription"
)

// ExpenseRunScope is the internal Go-struct mirror of ExpenseRecognitionRunScopeMsg.
type ExpenseRunScope struct {
	WorkspaceID            string
	SupplierID             string
	SupplierSubscriptionID string
	AsOfDate               string // YYYY-MM-DD; defaults to today (UTC)
}

// periodWindow holds the computed start/end for one billing period.
type periodWindow struct {
	Start string // YYYY-MM-DD
	End   string // YYYY-MM-DD
}

// ListExpenseRunCandidatesRepositories groups the cross-domain deps.
type ListExpenseRunCandidatesRepositories struct {
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	CostPlan             costplanpb.CostPlanDomainServiceServer
	ExpenseRecognition   expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
	Expenditure          expenditurepb.ExpenditureDomainServiceServer
}

// ListExpenseRunCandidatesServices groups infra services.
type ListExpenseRunCandidatesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListExpenseRunCandidatesUseCase enumerates pending recognition candidates
// for the given scope. Emits TWO source kinds:
//
//   - SUBSCRIPTION_CYCLE: active SupplierSubscription rows whose billing
//     cycles haven't been recognized as of AsOfDate.
//   - ADVANCE_DISBURSEMENT: active advance Disbursements (TIME_BASED, ACTIVE)
//     whose next tranche is due as of AsOfDate.
//
// Linked-advance suppression: for any subscription cycle, if there's an
// overlapping advance Disbursement for the same supplier+period, the
// subscription candidate is marked eligible=false with
// suppressing_advance_disbursement_id populated. The advance candidate
// still appears in the same list.
type ListExpenseRunCandidatesUseCase struct {
	repositories ListExpenseRunCandidatesRepositories
	services     ListExpenseRunCandidatesServices
}

// NewListExpenseRunCandidatesUseCase wires the use case.
func NewListExpenseRunCandidatesUseCase(
	repos ListExpenseRunCandidatesRepositories,
	svcs ListExpenseRunCandidatesServices,
) *ListExpenseRunCandidatesUseCase {
	return &ListExpenseRunCandidatesUseCase{repositories: repos, services: svcs}
}

// Execute returns the candidate list, with both source kinds merged.
func (uc *ListExpenseRunCandidatesUseCase) Execute(
	ctx context.Context,
	req *expenserecognitionrunpb.ListExpenseRunCandidatesRequest,
) (*expenserecognitionrunpb.ListExpenseRunCandidatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionCreate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierSubscription, ports.ActionRead); err != nil {
		return nil, err
	}

	scope := ExpenseRunScope{}
	if s := req.GetScope(); s != nil {
		scope.WorkspaceID = s.GetWorkspaceId()
		scope.SupplierID = s.GetSupplierId()
		scope.SupplierSubscriptionID = s.GetSupplierSubscriptionId()
		scope.AsOfDate = s.GetAsOfDate()
	}
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		scope.WorkspaceID = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}
	asOfDate := strings.TrimSpace(scope.AsOfDate)
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}
	asOfTime, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"expense_recognition.validation.invalid_as_of_date",
			"as_of_date must be YYYY-MM-DD [DEFAULT]",
		))
	}

	candidates := []*expenserecognitionrunpb.ExpenseRecognitionRunCandidate{}

	// Build subscription-cycle candidates first so the suppression lookup
	// can run against the advance set we discover next.
	subCandidates, subPeriodIndex, err := uc.buildSubscriptionCandidates(ctx, scope, asOfTime)
	if err != nil {
		return nil, err
	}

	advCandidates, advancesByCounterparty, err := uc.buildAdvanceCandidates(ctx, scope, asOfTime)
	if err != nil {
		return nil, err
	}

	// Apply linked-advance suppression to the subscription set.
	for _, c := range subCandidates {
		if !c.GetEligible() {
			continue
		}
		// Skip suppression when we couldn't resolve a supplier_id on the cycle.
		supplierID := c.GetSupplierId()
		if supplierID == "" {
			continue
		}
		ps := subPeriodIndex[c]
		if suppress, ok := findSuppressingAdvance(advancesByCounterparty[supplierID], ps); ok {
			c.Eligible = false
			c.BlockerReason = "suppressed_by_advance"
			suppressID := suppress
			c.SuppressingAdvanceDisbursementId = &suppressID
		}
	}

	candidates = append(candidates, subCandidates...)
	candidates = append(candidates, advCandidates...)

	limit := req.GetLimit()
	if limit > 0 && len(candidates) > int(limit) {
		nextCursor := fmt.Sprintf("%d", limit)
		return &expenserecognitionrunpb.ListExpenseRunCandidatesResponse{
			Data:       candidates[:limit],
			Success:    true,
			NextCursor: &nextCursor,
		}, nil
	}
	return &expenserecognitionrunpb.ListExpenseRunCandidatesResponse{
		Data:    candidates,
		Success: true,
	}, nil
}

// buildSubscriptionCandidates walks SupplierSubscription rows and enumerates
// pending billing-cycle windows. Returns a parallel map[candidate]periodWindow
// for use by suppression matching.
func (uc *ListExpenseRunCandidatesUseCase) buildSubscriptionCandidates(
	ctx context.Context,
	scope ExpenseRunScope,
	asOfTime time.Time,
) (
	[]*expenserecognitionrunpb.ExpenseRecognitionRunCandidate,
	map[*expenserecognitionrunpb.ExpenseRecognitionRunCandidate]periodWindow,
	error,
) {
	periodIndex := make(map[*expenserecognitionrunpb.ExpenseRecognitionRunCandidate]periodWindow)
	subs, err := uc.listSubscriptions(ctx, scope)
	if err != nil {
		return nil, periodIndex, err
	}
	existing := uc.buildExistingMarkerSet(ctx, subs)

	out := []*expenserecognitionrunpb.ExpenseRecognitionRunCandidate{}
	for _, sub := range subs {
		if !sub.GetActive() {
			continue
		}
		plan, err := uc.readCostPlan(ctx, sub.GetCostPlanId())
		if err != nil || plan == nil {
			continue
		}
		if !isCyclicForExpenseRun(plan) {
			continue
		}
		windows := enumeratePeriods(sub, plan, asOfTime)
		seen := existing[sub.GetId()]
		for _, w := range windows {
			marker := fmt.Sprintf("Period: %s → %s", w.Start, w.End)
			if seen[marker] {
				continue
			}
			subID := sub.GetId()
			supplierID := sub.GetSupplierId()
			c := &expenserecognitionrunpb.ExpenseRecognitionRunCandidate{
				SourceKind:             expenserecognitionrunpb.ExpenseRecognitionRunSourceKind_EXPENSE_RECOGNITION_RUN_SOURCE_KIND_SUBSCRIPTION_CYCLE,
				SupplierSubscriptionId: &subID,
				SupplierId:             supplierID,
				SupplierName:           supplierName(sub),
				SourceLabel:            sub.GetName(),
				Currency:               plan.GetBillingCurrency(),
				PeriodStart:            w.Start,
				PeriodEnd:              w.End,
				PeriodLabel:            fmt.Sprintf("%s – %s", w.Start, w.End),
				PeriodMarker:           marker,
				Amount:                 plan.GetBillingAmount(),
				Eligible:               true,
			}
			out = append(out, c)
			periodIndex[c] = w
		}
	}
	return out, periodIndex, nil
}

// buildAdvanceCandidates walks active TIME_BASED advance Disbursements and
// emits a candidate per next-due tranche. Returns the per-supplier index
// the suppression check uses.
func (uc *ListExpenseRunCandidatesUseCase) buildAdvanceCandidates(
	ctx context.Context,
	scope ExpenseRunScope,
	asOfTime time.Time,
) (
	[]*expenserecognitionrunpb.ExpenseRecognitionRunCandidate,
	map[string][]advancePeriod,
	error,
) {
	bySupplier := make(map[string][]advancePeriod)
	out := []*expenserecognitionrunpb.ExpenseRecognitionRunCandidate{}
	if uc.repositories.TreasuryDisbursement == nil {
		return out, bySupplier, nil
	}

	// advance_kind / advance_status are INTEGER enum columns (Plan B Phase 0
	// migration 20260517150000). Generic SQL StringFilter wraps the field in
	// LOWER(...) which produces an invalid predicate against an integer
	// column — use NumberFilter with the enum numeric values, mirroring the
	// selling-side pattern in
	// list_revenue_run_candidates.go § listActiveTimeBasedAdvances.
	filters := []*commonpb.TypedFilter{
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
	if scope.SupplierID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "supplier_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.SupplierID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}

	resp, err := uc.repositories.TreasuryDisbursement.ListDisbursements(ctx, &disbursementpb.ListDisbursementsRequest{
		Filters: &commonpb.FilterRequest{Filters: filters},
	})
	if err != nil {
		return out, bySupplier, err
	}
	if resp == nil {
		return out, bySupplier, nil
	}

	for _, adv := range resp.GetData() {
		if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_TIME_BASED {
			continue
		}
		if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE {
			continue
		}
		tranche, ok, err := amortizeschedule.ComputeNextDueTranche(amortizeschedule.Inputs{
			StartDate:       adv.GetAdvanceStartDate(),
			EndDate:         adv.GetAdvanceEndDate(),
			PeriodCount:     int(adv.GetAdvancePeriodCount()),
			PeriodUnit:      adv.GetAdvancePeriodUnit(),
			TotalAmount:     adv.GetAdvanceTotalAmount(),
			ProrationPolicy: protoProrationToHelper(adv.GetAdvanceProrationPolicy()),
			AsOfDate:        asOfTime.Format("2006-01-02"),
		})
		if err != nil || !ok {
			continue
		}
		advID := adv.GetId()
		supplierID := adv.GetSupplierId()
		marker := fmt.Sprintf("Period: %s → %s", tranche.PeriodStart, tranche.PeriodEnd)

		// Idempotency screen: skip advance candidates whose tranche is already
		// recognized (mirrors the subscription-cycle marker check).
		if uc.advanceAlreadyRecognized(ctx, advID, tranche.PeriodStart) {
			continue
		}

		c := &expenserecognitionrunpb.ExpenseRecognitionRunCandidate{
			SourceKind:            expenserecognitionrunpb.ExpenseRecognitionRunSourceKind_EXPENSE_RECOGNITION_RUN_SOURCE_KIND_ADVANCE_DISBURSEMENT,
			AdvanceDisbursementId: &advID,
			SupplierId:            supplierID,
			SupplierName:          "",
			SourceLabel:           adv.GetName(),
			Currency:              adv.GetCurrency(),
			PeriodStart:           tranche.PeriodStart,
			PeriodEnd:             tranche.PeriodEnd,
			PeriodLabel:           fmt.Sprintf("%s – %s", tranche.PeriodStart, tranche.PeriodEnd),
			PeriodMarker:          marker,
			Amount:                tranche.Amount,
			Eligible:              true,
		}
		out = append(out, c)
		if supplierID != "" {
			bySupplier[supplierID] = append(bySupplier[supplierID], advancePeriod{
				AdvanceID: advID,
				Start:     tranche.PeriodStart,
				End:       tranche.PeriodEnd,
			})
		}
	}

	return out, bySupplier, nil
}

// advancePeriod holds the data the suppression check needs.
type advancePeriod struct {
	AdvanceID string
	Start     string // YYYY-MM-DD
	End       string
}

// findSuppressingAdvance returns the advance disbursement ID whose period
// fully or partially overlaps the subscription window. Returns ok=false when
// no advance suppresses this cycle.
func findSuppressingAdvance(advances []advancePeriod, sub periodWindow) (string, bool) {
	if len(advances) == 0 {
		return "", false
	}
	for _, a := range advances {
		if periodsOverlap(a.Start, a.End, sub.Start, sub.End) {
			return a.AdvanceID, true
		}
	}
	return "", false
}

func periodsOverlap(aStart, aEnd, bStart, bEnd string) bool {
	if aStart == "" || aEnd == "" || bStart == "" || bEnd == "" {
		return false
	}
	return aStart <= bEnd && bStart <= aEnd
}

func (uc *ListExpenseRunCandidatesUseCase) listSubscriptions(
	ctx context.Context,
	scope ExpenseRunScope,
) ([]*suppliersubscriptionpb.SupplierSubscription, error) {
	if uc.repositories.SupplierSubscription == nil {
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
	if scope.SupplierID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "supplier_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.SupplierID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}
	if scope.SupplierSubscriptionID != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    scope.SupplierSubscriptionID,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}
	resp, err := uc.repositories.SupplierSubscription.ListSupplierSubscriptions(ctx, &suppliersubscriptionpb.ListSupplierSubscriptionsRequest{
		Filters: &commonpb.FilterRequest{Filters: filters},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.GetData(), nil
}

func (uc *ListExpenseRunCandidatesUseCase) readCostPlan(
	ctx context.Context,
	id string,
) (*costplanpb.CostPlan, error) {
	if id == "" || uc.repositories.CostPlan == nil {
		return nil, nil
	}
	resp, err := uc.repositories.CostPlan.ReadCostPlan(ctx, &costplanpb.ReadCostPlanRequest{
		Data: &costplanpb.CostPlan{Id: id},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// buildExistingMarkerSet builds a per-subscription lookup of the period
// markers already recognized. Mirrors the selling-side helper.
func (uc *ListExpenseRunCandidatesUseCase) buildExistingMarkerSet(
	ctx context.Context,
	subs []*suppliersubscriptionpb.SupplierSubscription,
) map[string]map[string]bool {
	out := make(map[string]map[string]bool)
	if uc.repositories.ExpenseRecognition == nil {
		return out
	}
	for _, sub := range subs {
		seen := make(map[string]bool)
		resp, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "supplier_subscription_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    sub.GetId(),
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
				},
			},
		})
		if err != nil || resp == nil {
			out[sub.GetId()] = seen
			continue
		}
		for _, r := range resp.GetData() {
			if r.GetNotes() == "" {
				continue
			}
			first := r.GetNotes()
			if idx := strings.Index(first, "\n"); idx >= 0 {
				first = first[:idx]
			}
			seen[strings.TrimSpace(first)] = true
		}
		out[sub.GetId()] = seen
	}
	return out
}

// advanceAlreadyRecognized — true when an ExpenseRecognition already drains
// this advance for the given period_start.
func (uc *ListExpenseRunCandidatesUseCase) advanceAlreadyRecognized(
	ctx context.Context,
	advanceID, periodStart string,
) bool {
	if uc.repositories.ExpenseRecognition == nil {
		return false
	}
	// Screen using advance_disbursement_id + period_start (heuristic). The
	// dispatcher in generate_expense_run uses the exact workspace-scoped
	// idempotency key when it actually inserts.
	resp, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "advance_disbursement_id",
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
		ps := r.GetPeriodStart()
		if ps == nil {
			continue
		}
		if ps.AsTime().UTC().Format("2006-01-02") == periodStart {
			return true
		}
	}
	return false
}

// enumeratePeriods reproduces the selling-side period walker for
// SupplierSubscription billing cycles. Cap last period at asOfDate.
func enumeratePeriods(
	sub *suppliersubscriptionpb.SupplierSubscription,
	plan *costplanpb.CostPlan,
	asOfDate time.Time,
) []periodWindow {
	if sub.GetDateTimeStart() == nil {
		return nil
	}
	startUTC := sub.GetDateTimeStart().AsTime().UTC()
	subStart := time.Date(startUTC.Year(), startUTC.Month(), startUTC.Day(), 0, 0, 0, 0, time.UTC)
	cycleValue, cycleUnit := resolveCycleParams(plan)
	if cycleValue <= 0 || cycleUnit == "" {
		return nil
	}
	var subEnd *time.Time
	if e := sub.GetDateTimeEnd(); e != nil {
		eu := e.AsTime().UTC()
		t := time.Date(eu.Year(), eu.Month(), eu.Day(), 0, 0, 0, 0, time.UTC)
		subEnd = &t
	}

	var windows []periodWindow
	periodStart := subStart
	for {
		if periodStart.After(asOfDate) {
			break
		}
		if subEnd != nil && periodStart.After(*subEnd) {
			break
		}
		nextStart := amortizeschedule.AddPeriod(periodStart, cycleValue, cycleUnit)
		periodEnd := nextStart.AddDate(0, 0, -1)
		if subEnd != nil && periodEnd.After(*subEnd) {
			periodEnd = *subEnd
		}
		if periodEnd.After(asOfDate) {
			periodEnd = asOfDate
		}
		windows = append(windows, periodWindow{
			Start: periodStart.Format("2006-01-02"),
			End:   periodEnd.Format("2006-01-02"),
		})
		periodStart = nextStart
	}
	return windows
}

// isCyclicForExpenseRun returns true for CostPlan billing kinds that the
// run engine should enumerate (RECURRING, CONTRACT with a cycle).
func isCyclicForExpenseRun(plan *costplanpb.CostPlan) bool {
	if plan == nil {
		return false
	}
	switch plan.GetBillingKind() {
	case costplanpb.CostPlanBillingKind_COST_PLAN_BILLING_KIND_RECURRING:
		return true
	case costplanpb.CostPlanBillingKind_COST_PLAN_BILLING_KIND_CONTRACT:
		return plan.GetBillingCycleValue() > 0 || plan.GetDurationValue() > 0
	default:
		return false
	}
}

func resolveCycleParams(plan *costplanpb.CostPlan) (int, string) {
	if plan == nil {
		return 0, ""
	}
	if v := plan.GetBillingCycleValue(); v > 0 {
		u := strings.ToLower(strings.TrimSpace(plan.GetBillingCycleUnit()))
		if u != "" {
			return int(v), u
		}
	}
	if v := plan.GetDurationValue(); v > 0 {
		u := strings.ToLower(strings.TrimSpace(plan.GetDurationUnit()))
		if u != "" {
			return int(v), u
		}
	}
	return 0, ""
}

func supplierName(sub *suppliersubscriptionpb.SupplierSubscription) string {
	if sub.GetSupplier() != nil {
		return sub.GetSupplier().GetName()
	}
	return ""
}

func protoProrationToHelper(p advancekindpb.AdvanceProrationPolicy) amortizeschedule.ProrationPolicy {
	switch p {
	case advancekindpb.AdvanceProrationPolicy_ADVANCE_PRORATION_POLICY_DAY_PRORATED:
		return amortizeschedule.ProrationPolicyDayProrated
	case advancekindpb.AdvanceProrationPolicy_ADVANCE_PRORATION_POLICY_NEXT_PERIOD_START:
		return amortizeschedule.ProrationPolicyNextPeriodStart
	default:
		return amortizeschedule.ProrationPolicyFullTranche
	}
}
