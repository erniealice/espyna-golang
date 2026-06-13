// Plan B Phase 7 — MILESTONE advance recognition (buying side).
//
// RecognizeMilestoneAdvanceDisbursement consumes one
// disbursement_supplier_billing_event junction row, emits a single
// ExpenseRecognition tied to that SupplierBillingEvent + the advance
// Disbursement, decrements the advance counters, and (if drained) flips
// advance_status to FULLY_AMORTIZED.
//
// Idempotency anchor: junction.expense_recognition_id. Once set, repeat
// calls SKIP. Mirror of the selling-side use case.
//
// See docs/plan/20260517-advance-cash-events/plan.md §"Phase 7" / §"MILESTONE
// recognize button" + docs/wiki/articles/advance-cash-events.md MILESTONE
// section.
package disbursement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	supplierbillingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_billing_event"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	junctionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_supplier_billing_event"
)

// RecognizeMilestoneAdvanceDisbursementRepositories groups the cross-domain
// deps.
//
// The Junction repo is the canonical idempotency anchor — its
// expense_recognition_id column is the source of truth for "did we already
// recognize this milestone?".
type RecognizeMilestoneAdvanceDisbursementRepositories struct {
	TreasuryDisbursement             disbursementpb.DisbursementDomainServiceServer
	ExpenseRecognition               expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	SupplierBillingEvent             supplierbillingeventpb.SupplierBillingEventDomainServiceServer
	DisbursementSupplierBillingEvent junctionpb.DisbursementSupplierBillingEventDomainServiceServer
}

// RecognizeMilestoneAdvanceDisbursementServices groups infra services.
type RecognizeMilestoneAdvanceDisbursementServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// RecognizeMilestoneAdvanceDisbursementUseCase mirrors the selling-side use
// case.
type RecognizeMilestoneAdvanceDisbursementUseCase struct {
	repositories RecognizeMilestoneAdvanceDisbursementRepositories
	services     RecognizeMilestoneAdvanceDisbursementServices
	update       *UpdateDisbursementUseCase // Q1-B routing
}

// NewRecognizeMilestoneAdvanceDisbursementUseCase wires the use case.
func NewRecognizeMilestoneAdvanceDisbursementUseCase(
	repos RecognizeMilestoneAdvanceDisbursementRepositories,
	svcs RecognizeMilestoneAdvanceDisbursementServices,
	update *UpdateDisbursementUseCase,
) *RecognizeMilestoneAdvanceDisbursementUseCase {
	return &RecognizeMilestoneAdvanceDisbursementUseCase{repositories: repos, services: svcs, update: update}
}

// Execute recognizes one MILESTONE tranche from the advance Disbursement.
//
// Mirror of the selling-side flow; see that file's doc for the step list.
func (uc *RecognizeMilestoneAdvanceDisbursementUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.RecognizeMilestoneAdvanceDisbursementRequest,
) (*disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse, error) {
	if req == nil {
		req = &disbursementpb.RecognizeMilestoneAdvanceDisbursementRequest{}
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTreasuryDisbursement,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "expense_recognition",
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryDisbursementId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.id_required",
			"treasury_disbursement_id is required [DEFAULT]",
		))
	}
	if strings.TrimSpace(req.GetSupplierBillingEventId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.supplier_billing_event_id_required",
			"supplier_billing_event_id is required [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

func (uc *RecognizeMilestoneAdvanceDisbursementUseCase) executeCore(
	ctx context.Context,
	req *disbursementpb.RecognizeMilestoneAdvanceDisbursementRequest,
) (*disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse, error) {
	// 1. Read + lock the advance Disbursement.
	readResp, err := uc.repositories.TreasuryDisbursement.ReadDisbursement(ctx, &disbursementpb.ReadDisbursementRequest{
		Data: &disbursementpb.Disbursement{Id: req.GetTreasuryDisbursementId()},
	})
	if err != nil {
		return milestoneErrored(err), err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.not_found",
			"treasury_disbursement not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	adv := readResp.GetData()[0]

	// 2. Validate advance kind/status.
	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_MILESTONE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.recognize_milestone_requires_milestone",
			"RecognizeMilestoneAdvance requires advance_kind=MILESTONE [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.recognize_requires_active",
			"RecognizeMilestoneAdvance requires advance_status=ACTIVE [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 3. Locate the junction row by (disbursement_id + supplier_billing_event_id).
	junction, err := uc.findJunction(ctx, req.GetTreasuryDisbursementId(), req.GetSupplierBillingEventId())
	if err != nil {
		return milestoneErrored(err), err
	}
	if junction == nil {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.junction_not_found",
			"disbursement_supplier_billing_event junction not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 4. Idempotency check FIRST — junction.expense_recognition_id is the
	// anchor.
	if existingID := strings.TrimSpace(junction.GetExpenseRecognitionId()); existingID != "" {
		conflict := existingID
		return &disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse{
			Outcome:                         advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			ConflictingExpenseRecognitionId: &conflict,
			NewRemainingAmount:              adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount:             adv.GetAdvanceRecognizedAmount(),
			NewStatus:                       adv.GetAdvanceStatus(),
			TrancheAmount:                   junction.GetTrancheAmount(),
		}, nil
	}

	// 5. Validate SupplierBillingEvent status = BILLED.
	beResp, err := uc.repositories.SupplierBillingEvent.ReadSupplierBillingEvent(ctx, &supplierbillingeventpb.ReadSupplierBillingEventRequest{
		Data: &supplierbillingeventpb.SupplierBillingEvent{Id: req.GetSupplierBillingEventId()},
	})
	if err != nil {
		return milestoneErrored(err), err
	}
	if beResp == nil || len(beResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"supplier_billing_event.errors.not_found",
			"supplier_billing_event not found [DEFAULT]",
		))
		return milestoneErrored(err), err
	}
	be := beResp.GetData()[0]
	if be.GetStatus() != supplierbillingeventpb.SupplierBillingEventStatus_SUPPLIER_BILLING_EVENT_STATUS_BILLED {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.supplier_billing_event_not_billed",
			"SupplierBillingEvent must be in BILLED status to recognize [DEFAULT]",
		))
		return milestoneErrored(err), err
	}

	// 6. INSERT ExpenseRecognition.
	tranche := junction.GetTrancheAmount()
	recID, err := uc.insertRecognition(ctx, adv, junction, req, tranche)
	if err != nil {
		return milestoneErrored(err), err
	}

	// 7. UPDATE junction.expense_recognition_id.
	erid := recID
	junction.ExpenseRecognitionId = &erid
	now := time.Now()
	dm := now.UnixMilli()
	junction.DateModified = &dm
	if _, err := uc.repositories.DisbursementSupplierBillingEvent.UpdateDisbursementSupplierBillingEvent(ctx, &junctionpb.UpdateDisbursementSupplierBillingEventRequest{
		Data: junction,
	}); err != nil {
		return milestoneErrored(err), err
	}

	// 8. UPDATE treasury_disbursement advance_* counters + status.
	newRemaining := adv.GetAdvanceRemainingAmount() - tranche
	if newRemaining < 0 {
		newRemaining = 0
	}
	newRecognized := adv.GetAdvanceRecognizedAmount() + tranche
	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE
	if newRemaining == 0 {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_AMORTIZED
	}

	adv.AdvanceRemainingAmount = &newRemaining
	adv.AdvanceRecognizedAmount = &newRecognized
	adv.AdvanceStatus = &newStatus
	dmStr := now.Format(time.RFC3339)
	adv.DateModified = &dm
	adv.DateModifiedString = &dmStr

	if _, err := uc.update.Execute(ctx, &disbursementpb.UpdateDisbursementRequest{
		Data: adv,
	}); err != nil {
		return milestoneErrored(err), err
	}

	return &disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse{
		Outcome:              advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED,
		ExpenseRecognitionId: &recID,
		NewRemainingAmount:   newRemaining,
		NewRecognizedAmount:  newRecognized,
		NewStatus:            newStatus,
		TrancheAmount:        tranche,
	}, nil
}

// findJunction returns the single disbursement_supplier_billing_event
// row matching (disbursement_id, supplier_billing_event_id), or nil if
// missing.
func (uc *RecognizeMilestoneAdvanceDisbursementUseCase) findJunction(
	ctx context.Context,
	disbursementID, supplierBillingEventID string,
) (*junctionpb.DisbursementSupplierBillingEvent, error) {
	if uc.repositories.DisbursementSupplierBillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.junction_repo_unavailable",
			"disbursement_supplier_billing_event repository is not configured [DEFAULT]",
		))
	}
	resp, err := uc.repositories.DisbursementSupplierBillingEvent.ListDisbursementSupplierBillingEvents(ctx, &junctionpb.ListDisbursementSupplierBillingEventsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "treasury_disbursement_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    disbursementID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "supplier_billing_event_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    supplierBillingEventID,
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
		if j.GetTreasuryDisbursementId() == disbursementID && j.GetSupplierBillingEventId() == supplierBillingEventID {
			return j, nil
		}
	}
	return nil, nil
}

// insertRecognition persists the new ExpenseRecognition row.
//
// Idempotency key shape: `{workspace_id}:ADVANCE_MILESTONE:{advance_id}:{supplier_billing_event_id}`.
// The (junction.expense_recognition_id) is the authoritative anchor; the key
// is the secondary defense if a parallel call sneaks past the read-modify-
// write sequence.
//
// NOTE: ExpenseRecognition.proto does NOT have a supplier_billing_event_id
// back-edge field; the linkage is preserved on the junction row. The Notes
// field carries "Milestone: {supplier_billing_event_id}" for observability.
func (uc *RecognizeMilestoneAdvanceDisbursementUseCase) insertRecognition(
	ctx context.Context,
	adv *disbursementpb.Disbursement,
	junction *junctionpb.DisbursementSupplierBillingEvent,
	req *disbursementpb.RecognizeMilestoneAdvanceDisbursementRequest,
	tranche int64,
) (string, error) {
	recID := uc.services.IDGenerator.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	advanceID := adv.GetId()
	supplierBillingEventID := junction.GetSupplierBillingEventId()
	supplierID := adv.GetSupplierId()
	wsID := req.GetWorkspaceId()

	recognitionDate := timestamppb.New(now)
	idempotencyKey := fmt.Sprintf("%s:ADVANCE_MILESTONE:%s:%s", wsID, advanceID, supplierBillingEventID)
	notes := fmt.Sprintf("Milestone: %s", supplierBillingEventID)

	rec := &expenserecognitionpb.ExpenseRecognition{
		Id:                    recID,
		WorkspaceId:           wsID,
		DateCreated:           &dc,
		DateCreatedString:     &dcStr,
		DateModified:          &dc,
		DateModifiedString:    &dcStr,
		Active:                true,
		Name:                  fmt.Sprintf("Advance milestone recognition %s", supplierBillingEventID),
		RecognitionDate:       recognitionDate,
		Currency:              adv.GetCurrency(),
		TotalAmount:           tranche,
		Status:                expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_POSTED,
		IdempotencyKey:        idempotencyKey,
		AdvanceDisbursementId: &advanceID,
		Notes:                 &notes,
	}
	if supplierID != "" {
		rec.SupplierId = &supplierID
	}
	if runID := req.GetRunId(); runID != "" {
		r := runID
		rec.RunId = &r
	}

	resp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: rec,
	})
	if err != nil {
		return "", fmt.Errorf("create expense_recognition: %w", err)
	}
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), nil
	}
	return recID, nil
}

// milestoneErrored wraps an error into the proto response.
func milestoneErrored(err error) *disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse {
	out := &disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse{
		Outcome: advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED,
	}
	if err != nil {
		msg := err.Error()
		out.Error = &commonpb.Error{Message: msg}
	}
	return out
}
