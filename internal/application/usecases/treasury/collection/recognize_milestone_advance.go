// Plan B Phase 7 — MILESTONE advance recognition (selling side).
//
// RecognizeMilestoneAdvanceCollection consumes one
// collection_billing_event junction row, emits a single Revenue tied
// to that BillingEvent + the advance Collection, decrements the advance
// counters, and (if drained) flips advance_status to FULLY_RECOGNIZED.
//
// Idempotency anchor: junction.revenue_id. Once set, repeat calls SKIP.
//
// See docs/plan/20260517-advance-cash-events/plan.md §"Phase 7" / §"MILESTONE
// recognize button" + docs/wiki/articles/advance-cash-events.md MILESTONE
// section.
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

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	junctionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_billing_event"
)

// RecognizeMilestoneAdvanceCollectionRepositories groups the cross-domain deps.
//
// The Junction repo is the canonical idempotency anchor — its revenue_id
// column is the source of truth for "did we already recognize this
// milestone?". BillingEvent is read to validate status == BILLED. Revenue is
// the write target.
type RecognizeMilestoneAdvanceCollectionRepositories struct {
	TreasuryCollection             collectionpb.CollectionDomainServiceServer
	Revenue                        revenuepb.RevenueDomainServiceServer
	BillingEvent                   billingeventpb.BillingEventDomainServiceServer
	CollectionBillingEvent junctionpb.CollectionBillingEventDomainServiceServer
}

// RecognizeMilestoneAdvanceCollectionServices groups infra services.
type RecognizeMilestoneAdvanceCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// RecognizeMilestoneAdvanceCollectionUseCase wires Plan B's MILESTONE
// recognition flow on the selling side.
type RecognizeMilestoneAdvanceCollectionUseCase struct {
	repositories RecognizeMilestoneAdvanceCollectionRepositories
	services     RecognizeMilestoneAdvanceCollectionServices
	update       *UpdateCollectionUseCase // Q1-B routing
}

// NewRecognizeMilestoneAdvanceCollectionUseCase wires the use case.
func NewRecognizeMilestoneAdvanceCollectionUseCase(
	repos RecognizeMilestoneAdvanceCollectionRepositories,
	svcs RecognizeMilestoneAdvanceCollectionServices,
	update *UpdateCollectionUseCase,
) *RecognizeMilestoneAdvanceCollectionUseCase {
	return &RecognizeMilestoneAdvanceCollectionUseCase{repositories: repos, services: svcs, update: update}
}

// Execute recognizes one MILESTONE tranche from the advance Collection.
//
// Flow (single tx — caller may pass a tx-bound ctx via TransactionService):
//  1. authcheck (treasury_collection:update + revenue:create).
//  2. Read + lock the advance Collection (SELECT FOR UPDATE — adapter honors
//     the active tx).
//  3. Look up the junction row by (collection_id + billing_event_id).
//  4. Validate: advance_kind == MILESTONE; advance_status == ACTIVE; junction
//     exists; junction.revenue_id IS NULL; billing_event.status == BILLED.
//  5. Idempotency: if junction.revenue_id already set → SKIPPED.
//  6. INSERT Revenue (status=posted, advance_collection_id, billing_event_id,
//     currency=collection.currency, total=tranche_amount).
//  7. UPDATE junction.revenue_id = new revenue id.
//  8. UPDATE collection.advance_remaining/recognized; flip status if drained.
func (uc *RecognizeMilestoneAdvanceCollectionUseCase) Execute(
	ctx context.Context,
	req *collectionpb.RecognizeMilestoneAdvanceCollectionRequest,
) (*collectionpb.RecognizeMilestoneAdvanceCollectionResponse, error) {
	if req == nil {
		req = &collectionpb.RecognizeMilestoneAdvanceCollectionRequest{}
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
	if strings.TrimSpace(req.GetBillingEventId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.validation.billing_event_id_required",
			"billing_event_id is required [DEFAULT]",
		))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var out *collectionpb.RecognizeMilestoneAdvanceCollectionResponse
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

func (uc *RecognizeMilestoneAdvanceCollectionUseCase) executeCore(
	ctx context.Context,
	req *collectionpb.RecognizeMilestoneAdvanceCollectionRequest,
) (*collectionpb.RecognizeMilestoneAdvanceCollectionResponse, error) {
	// 1. Read + lock the advance Collection.
	readResp, err := uc.repositories.TreasuryCollection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
		Data: &collectionpb.Collection{Id: req.GetTreasuryCollectionId()},
	})
	if err != nil {
		return milestoneErrored(err), err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.not_found",
			"treasury_collection not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	adv := readResp.GetData()[0]

	// 2. Validate advance kind/status.
	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_MILESTONE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.recognize_milestone_requires_milestone",
			"RecognizeMilestoneAdvance requires advance_kind=MILESTONE [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.recognize_requires_active",
			"RecognizeMilestoneAdvance requires advance_status=ACTIVE [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 3. Locate the junction row. We read inline via List with a composite
	// filter (collection_id + billing_event_id) rather than adding a new
	// read-by-pair adapter method. The (collection_id, billing_event_id)
	// pair identifies at most one junction in v1.
	junction, err := uc.findJunction(ctx, req.GetTreasuryCollectionId(), req.GetBillingEventId())
	if err != nil {
		return milestoneErrored(err), err
	}
	if junction == nil {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.junction_not_found",
			"collection_billing_event junction not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 4. Idempotency check FIRST — junction.revenue_id is the anchor.
	if existingID := strings.TrimSpace(junction.GetRevenueId()); existingID != "" {
		conflict := existingID
		return &collectionpb.RecognizeMilestoneAdvanceCollectionResponse{
			Outcome:              advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			ConflictingRevenueId: &conflict,
			NewRemainingAmount:   adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount:  adv.GetAdvanceRecognizedAmount(),
			NewStatus:            adv.GetAdvanceStatus(),
			TrancheAmount:        junction.GetTrancheAmount(),
		}, nil
	}

	// 5. Validate BillingEvent status = BILLED.
	beResp, err := uc.repositories.BillingEvent.ReadBillingEvent(ctx, &billingeventpb.ReadBillingEventRequest{
		Data: &billingeventpb.BillingEvent{Id: req.GetBillingEventId()},
	})
	if err != nil {
		return milestoneErrored(err), err
	}
	if beResp == nil || len(beResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"billing_event.errors.not_found",
			"billing_event not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	be := beResp.GetData()[0]
	if be.GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.billing_event_not_billed",
			"BillingEvent must be in BILLED status to recognize [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 6. INSERT Revenue.
	tranche := junction.GetTrancheAmount()
	revenueID, err := uc.insertRevenue(ctx, adv, junction, req, tranche)
	if err != nil {
		return milestoneErrored(err), err
	}

	// 7. UPDATE junction.revenue_id.
	rid := revenueID
	junction.RevenueId = &rid
	now := time.Now()
	dm := now.UnixMilli()
	junction.DateModified = &dm
	if _, err := uc.repositories.CollectionBillingEvent.UpdateCollectionBillingEvent(ctx, &junctionpb.UpdateCollectionBillingEventRequest{
		Data: junction,
	}); err != nil {
		return milestoneErrored(err), err
	}

	// 8. UPDATE treasury_collection advance_* counters + status.
	newRemaining := adv.GetAdvanceRemainingAmount() - tranche
	if newRemaining < 0 {
		newRemaining = 0
	}
	newRecognized := adv.GetAdvanceRecognizedAmount() + tranche
	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE
	if newRemaining == 0 {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED
	}

	adv.AdvanceRemainingAmount = &newRemaining
	adv.AdvanceRecognizedAmount = &newRecognized
	adv.AdvanceStatus = &newStatus
	dmStr := now.Format(time.RFC3339)
	adv.DateModified = &dm
	adv.DateModifiedString = &dmStr

	if _, err := uc.update.Execute(ctx, &collectionpb.UpdateCollectionRequest{
		Data: adv,
	}); err != nil {
		return milestoneErrored(err), err
	}

	return &collectionpb.RecognizeMilestoneAdvanceCollectionResponse{
		Outcome:             advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED,
		RevenueId:           &revenueID,
		NewRemainingAmount:  newRemaining,
		NewRecognizedAmount: newRecognized,
		NewStatus:           newStatus,
		TrancheAmount:       tranche,
	}, nil
}

// findJunction returns the single collection_billing_event row
// matching (collection_id, billing_event_id), or nil if missing. We use List
// with a composite filter — both columns are indexed.
func (uc *RecognizeMilestoneAdvanceCollectionUseCase) findJunction(
	ctx context.Context,
	collectionID, billingEventID string,
) (*junctionpb.CollectionBillingEvent, error) {
	if uc.repositories.CollectionBillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_collection.errors.junction_repo_unavailable",
			"collection_billing_event repository is not configured [DEFAULT]",
		))
	}
	resp, err := uc.repositories.CollectionBillingEvent.ListCollectionBillingEvents(ctx, &junctionpb.ListCollectionBillingEventsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "treasury_collection_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    collectionID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "billing_event_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    billingEventID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	for _, j := range resp.GetData() {
		if j.GetTreasuryCollectionId() == collectionID && j.GetBillingEventId() == billingEventID {
			return j, nil
		}
	}
	return nil, nil
}

// insertRevenue persists the new Revenue row. Carries the
// advance_collection_id + billing_event_id back-edges so downstream reports
// can join without crawling the junction.
func (uc *RecognizeMilestoneAdvanceCollectionUseCase) insertRevenue(
	ctx context.Context,
	adv *collectionpb.Collection,
	junction *junctionpb.CollectionBillingEvent,
	req *collectionpb.RecognizeMilestoneAdvanceCollectionRequest,
	tranche int64,
) (string, error) {
	revenueID := uc.services.IDService.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	advanceID := adv.GetId()
	billingEventID := junction.GetBillingEventId()
	clientID := adv.GetClientId()
	revenueDate := now.Format("2006-01-02")
	notes := fmt.Sprintf("Milestone: %s", billingEventID)

	rev := &revenuepb.Revenue{
		Id:                  revenueID,
		DateCreated:         &dc,
		DateCreatedString:   &dcStr,
		DateModified:        &dc,
		DateModifiedString:  &dcStr,
		Active:              true,
		ClientId:            clientID,
		RevenueDate:         &revenueDate,
		TotalAmount:         tranche,
		Currency:            adv.GetCurrency(),
		Status:              "posted",
		Notes:               &notes,
		AdvanceCollectionId: &advanceID,
		BillingEventId:      &billingEventID,
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

// milestoneErrored wraps an error into the proto response so the run engine can
// record it without losing post-state context.
func milestoneErrored(err error) *collectionpb.RecognizeMilestoneAdvanceCollectionResponse {
	out := &collectionpb.RecognizeMilestoneAdvanceCollectionResponse{
		Outcome: advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED,
	}
	if err != nil {
		msg := err.Error()
		out.Error = &commonpb.Error{Message: msg}
	}
	return out
}
