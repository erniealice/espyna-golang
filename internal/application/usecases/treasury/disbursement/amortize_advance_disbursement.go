// Package treasurydisbursement holds Plan B Phase 2 use cases for the
// buying-side Advance Cash Events flow (treasury_disbursement rows whose
// advance_kind != NONE).
//
// Mirror of treasury_collection/amortize_advance_collection.go — see plan
// docs/plan/20260517-advance-cash-events/plan.md §"Use cases" / §"Phase 2".
package disbursement

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
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const entityTreasuryDisbursement = "treasury_disbursement"

// AmortizeAdvanceDisbursementRepositories groups the cross-domain deps.
type AmortizeAdvanceDisbursementRepositories struct {
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
	ExpenseRecognition   expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// AmortizeAdvanceDisbursementServices groups infra services.
type AmortizeAdvanceDisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// AmortizeAdvanceDisbursementUseCase mirrors AmortizeAdvanceCollectionUseCase.
type AmortizeAdvanceDisbursementUseCase struct {
	repositories AmortizeAdvanceDisbursementRepositories
	services     AmortizeAdvanceDisbursementServices
	update       *UpdateDisbursementUseCase // Q1-B routing
}

// NewAmortizeAdvanceDisbursementUseCase wires the use case.
func NewAmortizeAdvanceDisbursementUseCase(
	repos AmortizeAdvanceDisbursementRepositories,
	svcs AmortizeAdvanceDisbursementServices,
	update *UpdateDisbursementUseCase,
) *AmortizeAdvanceDisbursementUseCase {
	return &AmortizeAdvanceDisbursementUseCase{repositories: repos, services: svcs, update: update}
}

// Execute amortizes one tranche from the advance Disbursement.
func (uc *AmortizeAdvanceDisbursementUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.AmortizeAdvanceDisbursementRequest,
) (*disbursementpb.AmortizeAdvanceDisbursementResponse, error) {
	if req == nil {
		req = &disbursementpb.AmortizeAdvanceDisbursementRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTreasuryDisbursement, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"expense_recognition", ports.ActionCreate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryDisbursementId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.validation.id_required",
			"treasury_disbursement_id is required [DEFAULT]",
		))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var out *disbursementpb.AmortizeAdvanceDisbursementResponse
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

func (uc *AmortizeAdvanceDisbursementUseCase) executeCore(
	ctx context.Context,
	req *disbursementpb.AmortizeAdvanceDisbursementRequest,
) (*disbursementpb.AmortizeAdvanceDisbursementResponse, error) {
	// 1. Read + lock the source row (FOR UPDATE inside the active tx).
	readResp, err := uc.repositories.TreasuryDisbursement.ReadDisbursement(ctx, &disbursementpb.ReadDisbursementRequest{
		Data: &disbursementpb.Disbursement{Id: req.GetTreasuryDisbursementId()},
	})
	if err != nil {
		return errored(err), err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.not_found",
			"treasury_disbursement not found [DEFAULT]",
		))
		return errored(err), err
	}
	adv := readResp.GetData()[0]

	// 2. Validate advance kind/status.
	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_TIME_BASED {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.amortize_requires_time_based",
			"AmortizeAdvanceDisbursement requires advance_kind=TIME_BASED [DEFAULT]",
		))
		return errored(err), err
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE {
		err := errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.amortize_requires_active",
			"AmortizeAdvanceDisbursement requires advance_status=ACTIVE [DEFAULT]",
		))
		return errored(err), err
	}

	// 3. Compute next-due tranche.
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
		ProrationPolicy: protoProrationToHelper(adv.GetAdvanceProrationPolicy()),
		AsOfDate:        asOf,
	})
	if err != nil {
		return errored(err), err
	}
	if !ok {
		return &disbursementpb.AmortizeAdvanceDisbursementResponse{
			Outcome:             advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			NewRemainingAmount:  adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount: adv.GetAdvanceRecognizedAmount(),
			NewStatus:           adv.GetAdvanceStatus(),
		}, nil
	}

	// 4. Idempotency check FIRST. Derive the canonical key + see if any
	// ExpenseRecognition already covers this tranche.
	idempotencyKey := req.GetIdempotencyKey()
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = BuildAdvanceIdempotencyKey(req.GetWorkspaceId(), req.GetTreasuryDisbursementId(), tranche.PeriodStart)
	}
	if conflictID, found, listErr := uc.findExistingRecognitionForKey(ctx, idempotencyKey, req.GetTreasuryDisbursementId(), tranche.PeriodStart); listErr != nil {
		return errored(listErr), listErr
	} else if found {
		conflict := conflictID
		return &disbursementpb.AmortizeAdvanceDisbursementResponse{
			Outcome:                         advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED,
			ConflictingExpenseRecognitionId: &conflict,
			NewRemainingAmount:              adv.GetAdvanceRemainingAmount(),
			NewRecognizedAmount:             adv.GetAdvanceRecognizedAmount(),
			NewStatus:                       adv.GetAdvanceStatus(),
			TrancheStart:                    tranche.PeriodStart,
			TrancheEnd:                      tranche.PeriodEnd,
			TrancheAmount:                   tranche.Amount,
		}, nil
	}

	// 5. INSERT ExpenseRecognition.
	recID, err := uc.insertRecognition(ctx, adv, tranche, req, idempotencyKey)
	if err != nil {
		return errored(err), err
	}

	// 6. UPDATE treasury_disbursement advance_* counters + status.
	newRemaining := adv.GetAdvanceRemainingAmount() - tranche.Amount
	if newRemaining < 0 {
		newRemaining = 0
	}
	newRecognized := adv.GetAdvanceRecognizedAmount() + tranche.Amount
	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE
	if newRemaining == 0 {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_AMORTIZED
	}

	adv.AdvanceRemainingAmount = &newRemaining
	adv.AdvanceRecognizedAmount = &newRecognized
	adv.AdvanceStatus = &newStatus
	now := time.Now()
	dm := now.UnixMilli()
	dmStr := now.Format(time.RFC3339)
	adv.DateModified = &dm
	adv.DateModifiedString = &dmStr

	if _, err := uc.update.Execute(ctx, &disbursementpb.UpdateDisbursementRequest{
		Data: adv,
	}); err != nil {
		return errored(err), err
	}

	return &disbursementpb.AmortizeAdvanceDisbursementResponse{
		Outcome:              advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED,
		ExpenseRecognitionId: &recID,
		NewRemainingAmount:   newRemaining,
		NewRecognizedAmount:  newRecognized,
		NewStatus:            newStatus,
		TrancheStart:         tranche.PeriodStart,
		TrancheEnd:           tranche.PeriodEnd,
		TrancheAmount:        tranche.Amount,
	}, nil
}

// findExistingRecognitionForKey looks for an existing ExpenseRecognition row
// either by exact idempotency_key match (preferred) or by
// (advance_disbursement_id + period_start) covering the same tranche.
//
// The DB enforces uniqueness on idempotency_key via a UNIQUE INDEX; this
// helper is the pre-INSERT check that gives the use case a SKIPPED outcome
// instead of an exception when the conflict exists.
func (uc *AmortizeAdvanceDisbursementUseCase) findExistingRecognitionForKey(
	ctx context.Context,
	key, advanceID, periodStart string,
) (string, bool, error) {
	if uc.repositories.ExpenseRecognition == nil {
		return "", false, nil
	}
	// Primary check — exact idempotency_key match (cheapest, hits the unique index).
	resp, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "idempotency_key",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    key,
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
	if resp != nil && len(resp.GetData()) > 0 {
		return resp.GetData()[0].GetId(), true, nil
	}

	// Secondary check — (advance_disbursement_id) filter, then defensive
	// period_start match against the in-memory rows. Guards against the case
	// where a previous recognition was inserted with a hand-rolled
	// idempotency_key that diverges from the canonical shape.
	resp2, err := uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, &expenserecognitionpb.ListExpenseRecognitionsRequest{
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
	if err != nil {
		return "", false, err
	}
	if resp2 == nil {
		return "", false, nil
	}
	for _, r := range resp2.GetData() {
		ps := r.GetPeriodStart()
		if ps == nil {
			continue
		}
		if ps.AsTime().UTC().Format("2006-01-02") == periodStart {
			return r.GetId(), true, nil
		}
	}
	return "", false, nil
}

// insertRecognition persists the new ExpenseRecognition row.
func (uc *AmortizeAdvanceDisbursementUseCase) insertRecognition(
	ctx context.Context,
	adv *disbursementpb.Disbursement,
	tranche amortizeschedule.TrancheSpec,
	req *disbursementpb.AmortizeAdvanceDisbursementRequest,
	idempotencyKey string,
) (string, error) {
	recID := uc.services.IDService.GenerateID()
	now := time.Now()
	dc := now.UnixMilli()
	dcStr := now.Format(time.RFC3339)

	ps, _ := time.Parse("2006-01-02", tranche.PeriodStart)
	pe, _ := time.Parse("2006-01-02", tranche.PeriodEnd)
	periodStart := timestamppb.New(ps)
	periodEnd := timestamppb.New(pe)
	recognitionDate := timestamppb.New(pe)

	advanceID := adv.GetId()
	supplierID := adv.GetSupplierId()
	wsID := req.GetWorkspaceId()

	rec := &expenserecognitionpb.ExpenseRecognition{
		Id:                    recID,
		WorkspaceId:           wsID,
		DateCreated:           &dc,
		DateCreatedString:     &dcStr,
		DateModified:          &dc,
		DateModifiedString:    &dcStr,
		Active:                true,
		Name:                  fmt.Sprintf("Advance amortization %s → %s", tranche.PeriodStart, tranche.PeriodEnd),
		RecognitionDate:       recognitionDate,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		Currency:              adv.GetCurrency(),
		TotalAmount:           tranche.Amount,
		Status:                expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_POSTED,
		IdempotencyKey:        idempotencyKey,
		AdvanceDisbursementId: &advanceID,
	}
	if supplierID != "" {
		rec.SupplierId = &supplierID
	}
	if runID := req.GetRunId(); runID != "" {
		r := runID
		rec.RunId = &r
	}
	notes := treasurycollectionPeriodMarker(tranche.PeriodStart, tranche.PeriodEnd)
	rec.Notes = &notes

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

// BuildAdvanceIdempotencyKey is the canonical key shape per plan:
//
//	{workspace_id}:ADVANCE:{advance_disbursement_id}:{period_start}
func BuildAdvanceIdempotencyKey(workspaceID, advanceID, periodStart string) string {
	return fmt.Sprintf("%s:ADVANCE:%s:%s", workspaceID, advanceID, periodStart)
}

// treasurycollectionPeriodMarker is the canonical period-marker for the
// notes field. Mirrors the selling-side BuildAdvancePeriodMarker but kept
// inline to avoid an import cycle between the two sibling packages.
func treasurycollectionPeriodMarker(start, end string) string {
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

func errored(err error) *disbursementpb.AmortizeAdvanceDisbursementResponse {
	out := &disbursementpb.AmortizeAdvanceDisbursementResponse{
		Outcome: advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED,
	}
	if err != nil {
		msg := err.Error()
		out.Error = &commonpb.Error{Message: msg}
	}
	return out
}
