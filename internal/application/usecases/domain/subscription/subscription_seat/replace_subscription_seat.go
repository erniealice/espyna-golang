package subscription_seat

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// ReplaceSubscriptionSeatRequest is the Go-shaped input for the SR-2 atomic
// seat replacement operation. (No proto request type exists for this op.)
//
// Identity-bearing fields (client_id / subscription_id / product_plan_id /
// position) are ALWAYS derived from the OLD seat, never from caller input.
type ReplaceSubscriptionSeatRequest struct {
	OldSeatID  string // the ACTIVE seat being replaced
	NewStaffID string // the incoming staff member

	EffectiveDate *int64 // optional date_start for the new seat (epoch millis)

	// NewContractedAmount overrides the carried-over amount ONLY when both it and
	// Authorization are present (SR: re-pricing a seat requires authorization).
	NewContractedAmount *int64
	Authorization       string

	// WorkRequestID, when present, makes the replace idempotent: a retry with the
	// same work_request_id is a no-op that returns the already-created new seat.
	WorkRequestID *string

	Reason *string // optional free-text audit note (not persisted in v1 — no column)
}

// ReplaceSubscriptionSeatRepositories groups all repository dependencies
type ReplaceSubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// seatRowLocker is the narrow optional interface the postgres seat adapter
// satisfies (LockSubscriptionSeatForUpdate). The replace use case type-asserts
// the seat repository to it so the OLD seat is read WITH a FOR UPDATE row lock
// inside the transaction — closing the SR-2 TOCTOU double-replace race. Adapters
// that do not implement it (mock/firestore) fall back to the unlocked read.
type seatRowLocker interface {
	LockSubscriptionSeatForUpdate(ctx context.Context, id string) (*subscriptionseatpb.SubscriptionSeat, error)
}

// ReplaceSubscriptionSeatServices groups all business service dependencies
type ReplaceSubscriptionSeatServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// ReplaceSubscriptionSeatUseCase performs the SR-2 atomic replace: it ends the old
// ACTIVE seat (status -> REPLACED + date_end) and creates a new ACTIVE seat in its
// position, in ONE transaction.
//
// The lock-the-old-row semantics: the whole read-old -> assert-active ->
// update-old -> insert-new sequence runs inside a single
// Transactor.ExecuteInTransaction (mirrors the transactional-orchestration
// pattern used by outcome_criteria.CreateOutcomeCriteria.executeWithTransaction).
// PRIMARY serialization is a SELECT ... FOR UPDATE row lock on the OLD seat
// (readOldSeatLocked): two concurrent replacers of the same OLD seat serialize on
// the lock — the second waits, then observes status=REPLACED and is rejected.
// This closure is independent of position. The partial UNIQUE
// (subscription_id, position) WHERE status='active' is a SECONDARY DB backstop for
// the same-position case; because Create now defaults a non-empty position
// (position_id = id for an original seat) it fires even when no caller supplied a
// position (NULL positions previously bypassed it — NULLs are distinct in a
// partial unique).
type ReplaceSubscriptionSeatUseCase struct {
	repositories ReplaceSubscriptionSeatRepositories
	services     ReplaceSubscriptionSeatServices
}

// NewReplaceSubscriptionSeatUseCase creates a new ReplaceSubscriptionSeatUseCase
func NewReplaceSubscriptionSeatUseCase(
	repositories ReplaceSubscriptionSeatRepositories,
	services ReplaceSubscriptionSeatServices,
) *ReplaceSubscriptionSeatUseCase {
	return &ReplaceSubscriptionSeatUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the atomic seat replacement.
func (uc *ReplaceSubscriptionSeatUseCase) Execute(ctx context.Context, req *ReplaceSubscriptionSeatRequest) (*subscriptionseatpb.SubscriptionSeat, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.SubscriptionSeat, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	if req == nil || req.OldSeatID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.id_required", "Subscription seat ID is required [DEFAULT]"))
	}
	if req.NewStaffID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.validation.staff_id_required", "Staff ID is required [DEFAULT]"))
	}
	// Re-pricing requires explicit authorization.
	if req.NewContractedAmount != nil && req.Authorization == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_seat.errors.reprice_requires_authorization", "Re-pricing a seat requires authorization [DEFAULT]"))
	}

	var newSeat *subscriptionseatpb.SubscriptionSeat

	run := func(txCtx context.Context) error {
		// Idempotency: if a seat already exists with this work_request_id, no-op.
		if req.WorkRequestID != nil && *req.WorkRequestID != "" {
			if existing, _ := uc.findActiveByWorkRequest(txCtx, *req.WorkRequestID); existing != nil {
				newSeat = existing
				return nil
			}
		}

		// Read + assert the OLD seat is ACTIVE. The read takes a FOR UPDATE row
		// lock (SR-2 TOCTOU defense): two concurrent replacers of the same OLD
		// seat serialize on the lock, so the second waits, then observes
		// status=REPLACED below and is rejected. This is the PRIMARY race closure
		// and does not depend on the partial-unique active-position backstop.
		old, err := uc.readOldSeatLocked(txCtx, req.OldSeatID)
		if err != nil {
			return err
		}
		if old == nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_seat.errors.not_found", "Subscription seat not found [DEFAULT]"))
		}
		if old.Status != subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_seat.errors.replace_requires_active", "Only an active seat can be replaced [DEFAULT]"))
		}

		now := time.Now()
		effective := now.UnixMilli()
		if req.EffectiveDate != nil {
			effective = *req.EffectiveDate
		}

		// End the OLD seat: status -> REPLACED + date_end. Done BEFORE the new
		// INSERT so the partial-unique active-position slot is freed for the
		// new ACTIVE seat.
		old.Status = subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_REPLACED
		old.Active = false
		old.DateEnd = &effective
		old.DateModified = &[]int64{now.UnixMilli()}[0]
		old.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		if _, err := uc.repositories.SubscriptionSeat.UpdateSubscriptionSeat(txCtx, &subscriptionseatpb.UpdateSubscriptionSeatRequest{Data: old}); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_seat.errors.update_failed", "Subscription seat update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// Build the NEW ACTIVE seat. Identity fields are derived from OLD, never
		// from caller input. contracted_amount carries over UNLESS an authorized
		// new amount was supplied.
		contracted := old.ContractedAmount
		if req.NewContractedAmount != nil && req.Authorization != "" {
			contracted = req.NewContractedAmount
		}
		newSeat = &subscriptionseatpb.SubscriptionSeat{
			Id:                 uc.services.IDGenerator.GenerateID(),
			WorkspaceId:        old.WorkspaceId,
			SubscriptionId:     old.SubscriptionId,   // derived from OLD
			StaffId:            req.NewStaffID,       // the only caller-driven identity
			ClientId:           old.ClientId,         // derived from OLD (IDOR denorm)
			ProductPlanId:      old.ProductPlanId,    // derived from OLD (billing anchor)
			ProductVariantId:   old.ProductVariantId, // derived from OLD
			ContractedAmount:   contracted,
			ContractedCurrency: old.ContractedCurrency,
			RoleTitle:          old.RoleTitle,
			Seniority:          old.Seniority,
			Position:           old.Position, // derived from OLD
			ReviewCadenceValue: old.ReviewCadenceValue,
			ReviewCadenceUnit:  old.ReviewCadenceUnit,
			ReplacesId:         &old.Id,
			WorkRequestId:      req.WorkRequestID,
			Status:             subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE,
			Active:             true,
			DateStart:          &effective,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		}
		if _, err := uc.repositories.SubscriptionSeat.CreateSubscriptionSeat(txCtx, &subscriptionseatpb.CreateSubscriptionSeatRequest{Data: newSeat}); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "subscription_seat.errors.creation_failed", "Subscription seat creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// SR-7: conditionally offboard the OLD staff member — but ONLY if they hold
		// no other ACTIVE seat anywhere. The check is performed here; the staff
		// offboard write itself is a future seam (no staff-status / work_request
		// entity exists in this plan to record the offboard against), so when
		// warranted we record intent without a mutation. Fail-open on the check is
		// avoided: an error querying other seats aborts the transaction.
		warranted, err := uc.offboardWarranted(txCtx, old.StaffId)
		if err != nil {
			return err
		}
		_ = warranted // future seam: trigger staff offboard when the entity lands

		return nil
	}

	// Wrap the whole sequence in a single transaction when supported.
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, run); err != nil {
			return nil, err
		}
		return newSeat, nil
	}
	if err := run(ctx); err != nil {
		return nil, err
	}
	return newSeat, nil
}

// readOldSeatLocked reads the OLD seat for replacement. When the seat repository
// supports row locking (postgres), it reads via SELECT ... FOR UPDATE inside the
// transaction (SR-2 serialization). Otherwise it falls back to the unlocked
// proto read (mock/firestore — no concurrent-write guarantee available there).
// Returns (nil, nil) when the seat does not exist.
func (uc *ReplaceSubscriptionSeatUseCase) readOldSeatLocked(ctx context.Context, oldSeatID string) (*subscriptionseatpb.SubscriptionSeat, error) {
	if locker, ok := uc.repositories.SubscriptionSeat.(seatRowLocker); ok {
		return locker.LockSubscriptionSeatForUpdate(ctx, oldSeatID)
	}
	readResp, err := uc.repositories.SubscriptionSeat.ReadSubscriptionSeat(ctx, &subscriptionseatpb.ReadSubscriptionSeatRequest{
		Data: &subscriptionseatpb.SubscriptionSeat{Id: oldSeatID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, nil
	}
	return readResp.Data[0], nil
}

// findActiveByWorkRequest returns the ACTIVE seat carrying the given
// work_request_id, if any (idempotency lookup). Returns (nil, nil) when none.
func (uc *ReplaceSubscriptionSeatUseCase) findActiveByWorkRequest(ctx context.Context, workRequestID string) (*subscriptionseatpb.SubscriptionSeat, error) {
	listResp, err := uc.repositories.SubscriptionSeat.ListSubscriptionSeats(ctx, &subscriptionseatpb.ListSubscriptionSeatsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{stringEq("work_request_id", workRequestID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if listResp == nil {
		return nil, nil
	}
	for _, s := range listResp.Data {
		if s.GetWorkRequestId() == workRequestID &&
			s.Status == subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE {
			return s, nil
		}
	}
	return nil, nil
}

// offboardWarranted reports whether the given staff member holds no other ACTIVE
// seat (so a workspace-wide offboard would be appropriate). The replaced seat is
// already REPLACED at call time, so it is excluded naturally.
func (uc *ReplaceSubscriptionSeatUseCase) offboardWarranted(ctx context.Context, staffID string) (bool, error) {
	if staffID == "" {
		return false, nil
	}
	listResp, err := uc.repositories.SubscriptionSeat.ListSubscriptionSeats(ctx, &subscriptionseatpb.ListSubscriptionSeatsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{stringEq("staff_id", staffID)},
		},
	})
	if err != nil {
		return false, err
	}
	if listResp == nil {
		return true, nil
	}
	for _, s := range listResp.Data {
		if s.Status == subscriptionseatpb.SubscriptionSeatStatus_SUBSCRIPTION_SEAT_STATUS_ACTIVE {
			return false, nil
		}
	}
	return true, nil
}

// stringEq builds a STRING_EQUALS TypedFilter for the given field.
func stringEq(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    value,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
}
