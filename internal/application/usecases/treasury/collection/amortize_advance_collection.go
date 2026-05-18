// Package treasurycollection holds Plan B Phase 2 use cases for the
// selling-side Advance Cash Events flow (treasury_collection rows whose
// advance_kind != NONE).
//
// See docs/plan/20260517-advance-cash-events/plan.md §"Use cases" / §"Phase 2".
package collection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	amortizeschedule "github.com/erniealice/espyna-golang/internal/application/shared/amortize_schedule"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

const entityTreasuryCollection = "treasury_collection"

// AmortizeAdvanceCollectionRepositories groups the cross-domain dependencies.
//
// Adapter ports required (proto-generated DomainServiceServer interfaces).
// The interfaces are sufficient for v1; if the adapter team needs additional
// helpers (e.g. SELECT FOR UPDATE, atomic decrement), they should add them as
// additional methods on the Collection / Revenue server interfaces — see the
// "Postgres adapter changes" section of the Advance Cash Events plan.
//
// The current implementation uses Read + Update for the row lock; the postgres
// adapter is expected to wrap the Update inside the active tx so that the
// SELECT FOR UPDATE semantics come for free via the TransactionService.
type AmortizeAdvanceCollectionRepositories struct {
	TreasuryCollection collectionpb.CollectionDomainServiceServer
	Revenue            revenuepb.RevenueDomainServiceServer
}

// AmortizeAdvanceCollectionServices groups infra services.
type AmortizeAdvanceCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// AmortizeAdvanceCollectionUseCase wires Plan B's selling-side amortization.
type AmortizeAdvanceCollectionUseCase struct {
	repositories AmortizeAdvanceCollectionRepositories
	services     AmortizeAdvanceCollectionServices
	// Q1-B (LOCKED): terminal row update goes through the wrapping use case.
	update *UpdateCollectionUseCase
}

// NewAmortizeAdvanceCollectionUseCase wires the use case with grouped deps.
func NewAmortizeAdvanceCollectionUseCase(
	repos AmortizeAdvanceCollectionRepositories,
	svcs AmortizeAdvanceCollectionServices,
	update *UpdateCollectionUseCase,
) *AmortizeAdvanceCollectionUseCase {
	return &AmortizeAdvanceCollectionUseCase{repositories: repos, services: svcs, update: update}
}

// Execute amortizes one tranche from the advance Collection.
//
// Flow (single tx — caller may pass a tx-bound ctx via TransactionService):
//  1. authcheck (treasury_collection:update + revenue:create).
//  2. SELECT FOR UPDATE the treasury_collection row (via repo Read; the
//     adapter is responsible for issuing FOR UPDATE inside the active tx).
//  3. Validate advance_kind = TIME_BASED + advance_status = ACTIVE.
//  4. Compute the next-due tranche via amortize_schedule.
//  5. Idempotency check FIRST — list existing Revenues for this advance and
//     bail with SKIPPED if any already cover this period_start.
//  6. INSERT Revenue (status=POSTED, advance_collection_id=…).
//  7. UPDATE treasury_collection advance_* counters + status.
func (uc *AmortizeAdvanceCollectionUseCase) Execute(
	ctx context.Context,
	req *collectionpb.AmortizeAdvanceCollectionRequest,
) (*collectionpb.AmortizeAdvanceCollectionResponse, error) {
	if req == nil {
		req = &collectionpb.AmortizeAdvanceCollectionRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTreasuryCollection, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"revenue", ports.ActionCreate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryCollectionId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.validation.id_required",
			"treasury_collection_id is required [DEFAULT]",
		))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var out *collectionpb.AmortizeAdvanceCollectionResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, execErr := uc.executeCore(txCtx, req)
			if execErr != nil {
				return execErr
			}
			out = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *AmortizeAdvanceCollectionUseCase) executeCore(
	ctx context.Context,
	req *collectionpb.AmortizeAdvanceCollectionRequest,
) (*collectionpb.AmortizeAdvanceCollectionResponse, error) {
	// 1. Read + lock the source row. The postgres adapter is expected to
	// honor an active tx and SELECT FOR UPDATE the row here. The mock and
	// firestore adapters degrade to plain reads — see plan §"Mock + Firestore
	// adapters" (DEFERRED for Phase 2).
	readResp, err := uc.repositories.TreasuryCollection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
		Data: &collectionpb.Collection{Id: req.GetTreasuryCollectionId()},
	})
	if err != nil {
		return errored(err), err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.not_found",
			"treasury_collection not found [DEFAULT]",
		))
		return errored(err), err
	}
	adv := readResp.GetData()[0]

	// 2. Validate advance kind/status.
	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_TIME_BASED {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.amortize_requires_time_based",
			"AmortizeAdvanceCollection requires advance_kind=TIME_BASED [DEFAULT]",
		))
		return errored(err), err
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.amortize_requires_active",
			"AmortizeAdvanceCollection requires advance_status=ACTIVE [DEFAULT]",
		))
		return errored(err), err
	}

	// 3. Compute the next-due tranche.
	asOf := req.GetAsOfDate()
	if strings.TrimSpace(asOf) == "" {
		asOf = time.Now().UTC().Format("2006-01-02")
	}
	tranche, ok, err := amortizeschedule.ComputeNextDueTranche(amortizeschedule.Inputs{
		StartDate:       adv.GetAdvanceStartDate(),
		EndDate:         adv.GetAdvanceEndDate(),
		PeriodCount:     int(adv.GetAdvancePeriodCount()),
		PeriodUnit:      adv.GetAdvancePeriodUnit(),
		TotalAmount:     adv.GetAdvanceTotalAmount(),
		ProrationPolicy: ProtoProrationToHelper(adv.GetAdvanceProrationPolicy()),
		AsOfDate:        asOf,
	})
	if err != nil {
		return errored(err), err
	}
	if !ok {
		// No tranche due as of this date — treat as SKIPPED for run aggregate.
		return &collectionpb.AmortizeAdvanceCollectionResponse{
			Outcome:             advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			NewRemainingAmount:  adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount: adv.GetAdvanceRecognizedAmount(),
			NewStatus:           adv.GetAdvanceStatus(),
		}, nil
	}

	// 4. Idempotency check FIRST. We look for any existing Revenue carrying
	// advance_collection_id == this advance AND period_marker covering the
	// computed tranche period_start. If found, we SKIP — DO NOT INSERT.
	if conflictID, found, listErr := uc.findExistingRevenueForPeriod(ctx, req.GetTreasuryCollectionId(), tranche.PeriodStart, tranche.PeriodEnd); listErr != nil {
		return errored(listErr), listErr
	} else if found {
		conflict := conflictID
		return &collectionpb.AmortizeAdvanceCollectionResponse{
			Outcome:              advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			ConflictingRevenueId: &conflict,
			NewRemainingAmount:   adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount:  adv.GetAdvanceRecognizedAmount(),
			NewStatus:            adv.GetAdvanceStatus(),
			TrancheStart:         tranche.PeriodStart,
			TrancheEnd:           tranche.PeriodEnd,
			TrancheAmount:        tranche.Amount,
		}, nil
	}

	// 5. INSERT Revenue.
	//
	// Concurrency safety net: the partial unique index
	// `idx_revenue_advance_period_unique` (migration 20260517180000) protects
	// against a second writer that races past the read-before-write check
	// above. When that fires, the postgres adapter wraps the violation as
	// `ErrPeriodAlreadyInvoiced`; we detect via the same `period_already_invoiced`
	// substring the subscription path uses and translate to a SKIPPED outcome
	// (NOT ERRORED) with `conflicting_revenue_id` populated by re-listing.
	revenueID, err := uc.insertRevenue(ctx, adv, tranche, req)
	if err != nil {
		if strings.Contains(err.Error(), "period_already_invoiced") {
			conflictID, _, _ := uc.findExistingRevenueForPeriod(
				ctx, req.GetTreasuryCollectionId(), tranche.PeriodStart, tranche.PeriodEnd,
			)
			resp := &collectionpb.AmortizeAdvanceCollectionResponse{
				Outcome:             advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
				NewRemainingAmount:  adv.GetAdvanceRemainingAmount(),
				NewRecognizedAmount: adv.GetAdvanceRecognizedAmount(),
				NewStatus:           adv.GetAdvanceStatus(),
				TrancheStart:        tranche.PeriodStart,
				TrancheEnd:          tranche.PeriodEnd,
				TrancheAmount:       tranche.Amount,
			}
			if conflictID != "" {
				resp.ConflictingRevenueId = &conflictID
			}
			return resp, nil
		}
		return errored(err), err
	}

	// 6. UPDATE treasury_collection advance_* counters + status.
	newRemaining := adv.GetAdvanceRemainingAmount() - tranche.Amount
	if newRemaining < 0 {
		newRemaining = 0
	}
	newRecognized := adv.GetAdvanceRecognizedAmount() + tranche.Amount
	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE
	if newRemaining == 0 {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED
	}

	adv.AdvanceRemainingAmount = &newRemaining
	adv.AdvanceRecognizedAmount = &newRecognized
	adv.AdvanceStatus = &newStatus
	now := time.Now()
	dm := now.UnixMilli()
	dmStr := now.Format(time.RFC3339)
	adv.DateModified = &dm
	adv.DateModifiedString = &dmStr

	// Q1-B (LOCKED) — route through wrapping UpdateCollection use case so
	// authcheck + ADVANCE_KIND_BURN_DOWN guard fire consistently. The wrapper
	// is transaction-aware (IsTransactionActive short-circuit) so this does
	// not start a nested independent transaction.
	if _, err := uc.update.Execute(ctx, &collectionpb.UpdateCollectionRequest{
		Data: adv,
	}); err != nil {
		return errored(err), err
	}

	return &collectionpb.AmortizeAdvanceCollectionResponse{
		Outcome:             advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED,
		RevenueId:           &revenueID,
		NewRemainingAmount:  newRemaining,
		NewRecognizedAmount: newRecognized,
		NewStatus:           newStatus,
		TrancheStart:        tranche.PeriodStart,
		TrancheEnd:          tranche.PeriodEnd,
		TrancheAmount:       tranche.Amount,
	}, nil
}

// findExistingRevenueForPeriod scans existing Revenues bound to this advance
// looking for one whose period_marker matches the computed tranche window.
//
// Idempotency anchor: the period marker — stored in Revenue.notes by
// buildAdvanceNotes — is the canonical "did we already recognize this period?"
// answer. The postgres adapter additionally enforces a UNIQUE INDEX on
// (advance_collection_id, period_marker) via the existing period_marker
// GENERATED column.
func (uc *AmortizeAdvanceCollectionUseCase) findExistingRevenueForPeriod(
	ctx context.Context,
	advanceID, periodStart, periodEnd string,
) (string, bool, error) {
	if uc.repositories.Revenue == nil {
		return "", false, nil
	}
	resp, err := uc.repositories.Revenue.ListRevenues(ctx, &revenuepb.ListRevenuesRequest{
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
	if err != nil {
		return "", false, err
	}
	if resp == nil {
		return "", false, nil
	}
	wantMarker := BuildAdvancePeriodMarker(periodStart, periodEnd)
	for _, r := range resp.GetData() {
		n := r.GetNotes()
		if n == "" {
			continue
		}
		// The marker is the first line of notes; extract up to first newline.
		first := n
		if idx := strings.Index(n, "\n"); idx >= 0 {
			first = n[:idx]
		}
		if strings.TrimSpace(first) == wantMarker {
			return r.GetId(), true, nil
		}
	}
	return "", false, nil
}

// insertRevenue persists the new Revenue row. The notes field gets the
// canonical period marker so the existing period_marker GENERATED column
// resolves to a non-null value.
func (uc *AmortizeAdvanceCollectionUseCase) insertRevenue(
	ctx context.Context,
	adv *collectionpb.Collection,
	tranche amortizeschedule.TrancheSpec,
	req *collectionpb.AmortizeAdvanceCollectionRequest,
) (string, error) {
	revenueID := uc.services.IDService.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	notes := BuildAdvanceNotes(tranche.PeriodStart, tranche.PeriodEnd)
	advanceID := adv.GetId()
	clientID := adv.GetClientId()
	revenueDate := tranche.PeriodEnd

	rev := &revenuepb.Revenue{
		Id:                  revenueID,
		DateCreated:         &dc,
		DateCreatedString:   &dcStr,
		DateModified:        &dc,
		DateModifiedString:  &dcStr,
		Active:              true,
		ClientId:            clientID,
		RevenueDate:         &revenueDate,
		TotalAmount:         tranche.Amount,
		Currency:            adv.GetCurrency(),
		Status:              "posted",
		Notes:               &notes,
		AdvanceCollectionId: &advanceID,
	}
	if runID := req.GetRunId(); runID != "" {
		r := runID
		rev.RunId = &r
	}

	resp, err := uc.repositories.Revenue.CreateRevenue(ctx, &revenuepb.CreateRevenueRequest{Data: rev})
	if err != nil {
		return "", fmt.Errorf("create revenue: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), nil
	}
	return revenueID, nil
}

// BuildAdvancePeriodMarker is the canonical encoding for the first-line of
// Revenue.notes carrying the period marker. Re-uses the same shape as the
// subscription-recognize flow ("Period: YYYY-MM-DD → YYYY-MM-DD") so the
// period_marker GENERATED column matches across both paths.
func BuildAdvancePeriodMarker(start, end string) string {
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

// BuildAdvanceNotes returns the Revenue.notes body — the canonical marker
// on the first line, no trailing whitespace.
func BuildAdvanceNotes(start, end string) string {
	return BuildAdvancePeriodMarker(start, end)
}

// ProtoProrationToHelper translates the proto enum to the helper's policy.
func ProtoProrationToHelper(p advancekindpb.AdvanceProrationPolicy) amortizeschedule.ProrationPolicy {
	switch p {
	case advancekindpb.AdvanceProrationPolicy_ADVANCE_PRORATION_POLICY_DAY_PRORATED:
		return amortizeschedule.ProrationPolicyDayProrated
	case advancekindpb.AdvanceProrationPolicy_ADVANCE_PRORATION_POLICY_NEXT_PERIOD_START:
		return amortizeschedule.ProrationPolicyNextPeriodStart
	default:
		return amortizeschedule.ProrationPolicyFullTranche
	}
}

// errored wraps an error into the proto response so the run engine can record
// it in the attempt accumulator without losing the post-state context.
func errored(err error) *collectionpb.AmortizeAdvanceCollectionResponse {
	out := &collectionpb.AmortizeAdvanceCollectionResponse{
		Outcome: advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED,
	}
	if err != nil {
		msg := err.Error()
		out.Error = &commonpb.Error{Message: msg}
	}
	return out
}
