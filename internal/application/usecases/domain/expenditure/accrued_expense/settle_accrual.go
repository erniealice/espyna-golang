// Package accruedexpense contains the use cases for the AccruedExpense
// entity (recognised supplier obligation ahead of bill arrival).
//
// settle_accrual.go — HIGH-3 single-write boundary for parent
// `settled_amount` / `remaining_amount` plus AccruedExpenseSettlement
// authoring.
//
// Why a single-write boundary
// ---------------------------
// The original SPS design carried `settling_expenditure_id` as a single
// FK on AccruedExpense — codex HIGH-3 rejected it because it can't
// model two partial bills settling one accrual, one bill spanning
// multiple accruals, or FX-adjusted settlements. The replacement is the
// AccruedExpenseSettlement join table (proto §6.4a). To keep parent
// totals consistent, ALL writes to AccruedExpense.settled_amount /
// remaining_amount / status now flow through this use case. Other use
// cases (Expenditure post, ExpenseRecognition, manual adjustment) MUST
// invoke SettleAccrual; they MUST NOT write to those fields directly.
// The reverse path (`reverse_accrual.go`, sister-agent owned) is the
// only other writer and only sets status=REVERSED.
//
// Transaction strategy
// --------------------
// The use case wraps the entire (validate -> insert settlement -> read
// peer settlements -> recompute parent -> update parent) sequence in a
// single transaction via the platform Transactor. The adapter
// layer is expected to take a row-level lock on the parent
// AccruedExpense row (`SELECT ... FOR UPDATE`) before reading peer
// settlements so concurrent SettleAccrual calls for the same accrual
// serialize. This guards against the lost-update race when two
// settlements land in the same instant.
//
// Idempotency
// -----------
// Settling the same Expenditure against the same AccruedExpense twice
// is a duplicate. We enforce it on `(accrued_expense_id,
// expenditure_id)` (DB unique index in migration
// `20260430140300_accrued_expense.up.sql`); for a graceful caller
// experience we ALSO short-circuit before INSERT: list existing
// settlements for the (accrued, expenditure) tuple inside the tx and
// return the existing row when found. The combined effect is "exactly
// one settlement row per (accrual, expenditure)" with retries returning
// the same id.
//
// Status transitions
// ------------------
//
//	OUTSTANDING                       — sum(settlements) == 0
//	OUTSTANDING -> PARTIAL            — 0 < sum(settlements) < accrued
//	PARTIAL     -> SETTLED            — sum(settlements) == accrued
//	any         -> REVERSED           — only by reverse_accrual.go
//
// Negative amounts and over-settlement are rejected: settling more than
// the remaining amount is an error (the AP team must adjust the parent
// estimate via a separate reverse-and-recreate workflow).
package accruedexpense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// entityAccruedExpenseSettle is the authcheck entity key. Local to
// this file to avoid duplicate-symbol conflicts with the sister
// agent's create.go which declares `entityAccruedExpense`.
const entityAccruedExpenseSettle = "accrued_expense"

// SettleAccrualRepositories groups repository dependencies. The same
// proto service satisfies both AccruedExpense CRUD and
// AccruedExpenseSettlement CRUD (separate gRPC services in the proto;
// the postgres adapter implements both). The settlement service is
// declared explicitly to keep the dependency graph honest.
type SettleAccrualRepositories struct {
	AccruedExpense           accruedexpensepb.AccruedExpenseDomainServiceServer
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// SettleAccrualServices groups service dependencies.
type SettleAccrualServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// SettleAccrualUseCase is the SOLE writer of
// AccruedExpense.settled_amount / remaining_amount / (PARTIAL|SETTLED)
// status. See package-doc for the boundary rules.
type SettleAccrualUseCase struct {
	repositories SettleAccrualRepositories
	services     SettleAccrualServices
}

// NewSettleAccrualUseCase wires dependencies.
func NewSettleAccrualUseCase(
	repositories SettleAccrualRepositories,
	services SettleAccrualServices,
) *SettleAccrualUseCase {
	return &SettleAccrualUseCase{repositories: repositories, services: services}
}

// Execute is the contracts.NewGenericHandler shim. SettleAccrual is the
// domain-named entry point; Execute exists so the use case satisfies the
// generic handler contract used by espyna's route registration.
//
// 2026-04-30 supplier-pricing-symmetry plan — SPS Wave 2 route hole closed.
func (uc *SettleAccrualUseCase) Execute(
	ctx context.Context,
	req *accruedexpensepb.SettleAccrualRequest,
) (*accruedexpensepb.SettleAccrualResponse, error) {
	return uc.SettleAccrual(ctx, req)
}

// SettleAccrual posts a settlement of `amount_settled` (centavos) from
// the given Expenditure against the given AccruedExpense and updates
// the parent's settled_amount / remaining_amount / status atomically.
//
// Idempotency: a duplicate (accrued_expense_id, expenditure_id) tuple
// returns the existing settlement and the parent unchanged.
func (uc *SettleAccrualUseCase) SettleAccrual(
	ctx context.Context,
	request *accruedexpensepb.SettleAccrualRequest,
) (*accruedexpensepb.SettleAccrualResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAccruedExpenseSettle, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if err := uc.validateRequest(ctx, request); err != nil {
		return nil, err
	}

	// Always run inside a transaction. Without it the read-modify-write
	// on the parent row races concurrent settlements for the same
	// accrual.
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *accruedexpensepb.SettleAccrualResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, request)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w",
				contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
					"accrued_expense.errors.settle_failed",
					"[ERR-DEFAULT] Failed to settle accrued expense"), err)
		}
		return result, nil
	}
	// No-tx fallback for mock/test environments. The race window is
	// real but the platform contracts guarantee Transactor is
	// always supplied in production.
	return uc.executeCore(ctx, request)
}

// validateRequest performs the cheap input checks that don't require
// the database. Rejects nil request, missing ids, non-positive
// amounts, and missing currency.
func (uc *SettleAccrualUseCase) validateRequest(
	ctx context.Context,
	request *accruedexpensepb.SettleAccrualRequest,
) error {
	if request == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.settle_request_required",
			"Settle accrual request is required [DEFAULT]"))
	}
	if request.GetAccruedExpenseId() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.id_required",
			"Accrued expense ID is required [DEFAULT]"))
	}
	if request.GetExpenditureId() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.expenditure_id_required",
			"Settling expenditure ID is required [DEFAULT]"))
	}
	if request.GetAmountSettled() <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.amount_positive",
			"Settlement amount must be greater than zero [DEFAULT]"))
	}
	if request.GetCurrency() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.validation.currency_required",
			"Settlement currency is required [DEFAULT]"))
	}
	return nil
}

// executeCore is the in-tx body. It expects to be called inside a
// transaction so the read-then-write window on the parent row is
// safely serialised by the adapter's row lock.
func (uc *SettleAccrualUseCase) executeCore(
	ctx context.Context,
	request *accruedexpensepb.SettleAccrualRequest,
) (*accruedexpensepb.SettleAccrualResponse, error) {
	// Read parent first. If it doesn't exist or has been REVERSED we
	// reject before inserting any settlement row.
	parentResp, err := uc.repositories.AccruedExpense.ReadAccruedExpense(ctx, &accruedexpensepb.ReadAccruedExpenseRequest{
		Data: &accruedexpensepb.AccruedExpense{Id: request.GetAccruedExpenseId()},
	})
	if err != nil {
		return nil, fmt.Errorf("read accrued expense %s: %w", request.GetAccruedExpenseId(), err)
	}
	parents := parentResp.GetData()
	if len(parents) == 0 || parents[0] == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.errors.not_found",
			"Accrued expense not found [DEFAULT]"))
	}
	parent := parents[0]

	if parent.GetStatus() == accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_REVERSED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.errors.cannot_settle_reversed",
			"Cannot settle a reversed accrual [DEFAULT]"))
	}
	if parent.GetCurrency() != "" && parent.GetCurrency() != request.GetCurrency() &&
		request.GetFxRate() == 0 {
		// Cross-currency settlement is allowed only with an explicit
		// FX rate. Without one we cannot recompute the parent's
		// settled_amount in its native currency.
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"accrued_expense.errors.fx_rate_required",
			"FX rate is required when settling in a different currency [DEFAULT]"))
	}

	// Idempotency check: existing (accrued, expenditure) pair short-
	// circuits with the existing settlement.
	existingResp, err := uc.repositories.AccruedExpenseSettlement.ListAccruedExpenseSettlements(ctx,
		&accruedexpensepb.ListAccruedExpenseSettlementsRequest{
			AccruedExpenseId: stringPtr(parent.GetId()),
			ExpenditureId:    stringPtr(request.GetExpenditureId()),
		})
	if err != nil {
		return nil, fmt.Errorf("list settlements for idempotency check: %w", err)
	}
	for _, existing := range existingResp.GetData() {
		if existing == nil {
			continue
		}
		// Skip reversal rows (they reference the original via
		// reversed_by_settlement_id) — only the original counts as
		// the dedup row.
		if existing.GetReversedBySettlementId() != "" {
			continue
		}
		if existing.GetExpenditureId() == request.GetExpenditureId() {
			return &accruedexpensepb.SettleAccrualResponse{
				AccruedExpense: parent,
				Settlement:     existing,
				Success:        true,
			}, nil
		}
	}

	// Recompute the prospective settled_amount under the new
	// settlement and reject over-settlement before any write.
	allResp, err := uc.repositories.AccruedExpenseSettlement.ListAccruedExpenseSettlements(ctx,
		&accruedexpensepb.ListAccruedExpenseSettlementsRequest{
			AccruedExpenseId: stringPtr(parent.GetId()),
		})
	if err != nil {
		return nil, fmt.Errorf("list peer settlements: %w", err)
	}
	priorSettled := sumActiveSettlements(allResp.GetData())
	prospective := priorSettled + request.GetAmountSettled()
	if prospective > parent.GetAccruedAmount() {
		return nil, fmt.Errorf("settlement of %d exceeds remaining accrual %d (accrued=%d, already_settled=%d)",
			request.GetAmountSettled(),
			parent.GetAccruedAmount()-priorSettled,
			parent.GetAccruedAmount(),
			priorSettled,
		)
	}

	// Insert the settlement row.
	now := time.Now().UTC()
	settlement := &accruedexpensepb.AccruedExpenseSettlement{
		Id:                 uc.generateID(),
		WorkspaceId:        parent.GetWorkspaceId(),
		Active:             true,
		AccruedExpenseId:   parent.GetId(),
		ExpenditureId:      request.GetExpenditureId(),
		AmountSettled:      request.GetAmountSettled(),
		Currency:           request.GetCurrency(),
		SettledAt:          timestampPB(now),
		DateCreated:        int64Ptr(now.UnixMilli()),
		DateCreatedString:  stringPtr(now.Format(time.RFC3339)),
		DateModified:       int64Ptr(now.UnixMilli()),
		DateModifiedString: stringPtr(now.Format(time.RFC3339)),
	}
	if request.GetExpenditureLineItemId() != "" {
		v := request.GetExpenditureLineItemId()
		settlement.ExpenditureLineItemId = &v
	}
	if request.GetFxRate() != 0 {
		v := request.GetFxRate()
		settlement.FxRate = &v
	}
	if request.GetFxAdjustmentAmount() != 0 {
		v := request.GetFxAdjustmentAmount()
		settlement.FxAdjustmentAmount = &v
	}

	createResp, err := uc.repositories.AccruedExpenseSettlement.CreateAccruedExpenseSettlement(ctx,
		&accruedexpensepb.CreateAccruedExpenseSettlementRequest{Data: settlement})
	if err != nil {
		return nil, fmt.Errorf("create settlement row: %w", err)
	}
	if data := createResp.GetData(); len(data) > 0 && data[0] != nil {
		settlement = data[0]
	}

	// Recompute parent totals from the source-of-truth join rows. We
	// explicitly re-list (rather than priorSettled+amount) so any
	// concurrent reversal that landed inside our tx window is
	// reflected — the row lock + serialisation guarantee makes that
	// "free" but the explicit re-read matches the single-write-
	// boundary contract.
	finalResp, err := uc.repositories.AccruedExpenseSettlement.ListAccruedExpenseSettlements(ctx,
		&accruedexpensepb.ListAccruedExpenseSettlementsRequest{
			AccruedExpenseId: stringPtr(parent.GetId()),
		})
	if err != nil {
		return nil, fmt.Errorf("re-list settlements after insert: %w", err)
	}
	newSettled := sumActiveSettlements(finalResp.GetData())
	newRemaining := parent.GetAccruedAmount() - newSettled

	parent.SettledAmount = newSettled
	parent.RemainingAmount = newRemaining
	parent.Status = nextStatus(parent.GetStatus(), parent.GetAccruedAmount(), newSettled)
	parent.DateModified = int64Ptr(now.UnixMilli())
	parent.DateModifiedString = stringPtr(now.Format(time.RFC3339))

	updateResp, err := uc.repositories.AccruedExpense.UpdateAccruedExpense(ctx,
		&accruedexpensepb.UpdateAccruedExpenseRequest{Data: parent})
	if err != nil {
		return nil, fmt.Errorf("update accrued expense parent: %w", err)
	}
	if data := updateResp.GetData(); len(data) > 0 && data[0] != nil {
		parent = data[0]
	}

	return &accruedexpensepb.SettleAccrualResponse{
		AccruedExpense: parent,
		Settlement:     settlement,
		Success:        true,
	}, nil
}

// sumActiveSettlements adds AmountSettled across rows that have not
// been reversed. A reversal row carries `reversed_by_settlement_id`
// pointing at itself's parent settlement and a negative AmountSettled
// (per proto comment), so summing all `amount_settled` correctly nets
// reversals to zero. We exclude rows where Active=false to honour the
// soft-delete convention.
func sumActiveSettlements(rows []*accruedexpensepb.AccruedExpenseSettlement) int64 {
	var total int64
	for _, row := range rows {
		if row == nil || !row.GetActive() {
			continue
		}
		total += row.GetAmountSettled()
		if row.GetFxAdjustmentAmount() != 0 {
			total += row.GetFxAdjustmentAmount()
		}
	}
	return total
}

// nextStatus computes the parent status from the new settled total.
// REVERSED is sticky and only set by reverse_accrual.go.
func nextStatus(
	current accruedexpensepb.AccruedExpenseStatus,
	accrued int64,
	settled int64,
) accruedexpensepb.AccruedExpenseStatus {
	if current == accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_REVERSED {
		return current
	}
	if settled <= 0 {
		return accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_OUTSTANDING
	}
	if settled >= accrued {
		return accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_SETTLED
	}
	return accruedexpensepb.AccruedExpenseStatus_ACCRUED_EXPENSE_STATUS_PARTIAL
}

// generateID returns a fresh id, falling back to an empty string when
// the IDGenerator is not wired (the adapter is then responsible for
// assigning ids).
func (uc *SettleAccrualUseCase) generateID() string {
	if uc.services.IDGenerator == nil {
		return ""
	}
	return uc.services.IDGenerator.GenerateID()
}

// --- small helpers for proto optional fields -------------------------

func stringPtr(s string) *string                     { v := s; return &v }
func int64Ptr(i int64) *int64                        { v := i; return &v }
func timestampPB(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
