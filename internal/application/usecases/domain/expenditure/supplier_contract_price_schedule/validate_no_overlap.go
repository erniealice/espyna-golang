// Package suppliercontractpriceschedule contains the use cases for the
// SupplierContractPriceSchedule entity.
//
// validate_no_overlap.go — defense-in-depth overlap detection AND
// CRIT-2 atomic-truncate authoring rule (the entry point sister
// agents call from create.go / update.go).
//
// Why this validator exists when Postgres already enforces overlap
// ----------------------------------------------------------------
// The Postgres migration installs a GIST exclusion constraint
// (`supplier_contract_price_schedule_no_overlap`, see migration
// `20260430140000_supplier_contract_price_schedule.up.sql`) over
// `tstzrange(date_time_start, COALESCE(date_time_end, +inf), '[)')`
// scoped to non-cancelled rows. That is the FINAL guard — it
// rejects any commit that would leave overlapping rows for a given
// `supplier_contract_id`. This use-case-layer validator catches the
// same condition EARLIER with a friendlier error that names the
// conflicting row id and window. Surfacing the conflict before the
// INSERT means we never log a low-level `ExclusionViolation`, never
// burn an autoincrement / id, and the operator gets a message they
// can act on.
//
// The DB-side guarantee is also dialect-specific (GIST is Postgres
// only). On MySQL / SQL Server the application-side validator is
// the primary defense; per [plan.md §6.1] those dialects rely on a
// transaction-scoped advisory lock per `supplier_contract_id` plus
// this validator. The validator therefore must work correctly even
// when no DB exclusion constraint exists.
//
// Window semantics
// ----------------
//   - All windows are half-open `[date_time_start, date_time_end)`
//     with `date_time_end IS NULL` representing an open-ended
//     trailing window.
//   - All timestamps are normalised to UTC at the validator
//     boundary (display layer renders in workspace TZ; storage is
//     always UTC).
//   - The CANCELLED status is excluded from the comparison set —
//     cancelled rows never went into effect and must not block
//     creates.
//   - All non-cancelled statuses (SCHEDULED, ACTIVE, SUPERSEDED) are
//     considered. Historical inserts (a SUPERSEDED-class row) are
//     validated against ALL non-cancelled rows for the contract per
//     CRIT-2.
//
// Open-ended-row authoring rule (CRIT-2)
// --------------------------------------
// When the candidate row has `date_time_end IS NULL` AND a prior
// open-ended row exists for the same contract, the use case MUST
// atomically truncate the prior row's `date_time_end` to the
// candidate's `date_time_start` inside a single transaction. The
// validator handles the truncate IN-LINE: it issues UPDATE on the
// prior row before returning. The caller is already inside a
// `Transactor.ExecuteInTransaction` envelope (per
// `create.go` / `update.go`), so the UPDATE and the subsequent
// INSERT-of-candidate commit together or roll back together.
//
// Lock / serialisation strategy
// -----------------------------
// The validator does not invoke any explicit `FOR UPDATE` itself;
// instead it depends on the postgres adapter's `ListSupplierContractPriceSchedules`
// query running inside the open transaction. The adapter file owns
// the dialect-specific lock strategy:
//
//   - Postgres: the GIST exclusion constraint is the lock-free
//     correctness anchor — a racing INSERT that would overlap is
//     refused at COMMIT regardless of read-side ordering. The
//     validator's purpose is friendly errors, not concurrency.
//   - MySQL / SQL Server: the adapter takes a transaction-scoped
//     advisory lock keyed on `supplier_contract_id` BEFORE
//     servicing the List call so the read-then-write window is
//     closed at the application layer (no GIST equivalent).
//
// The validator therefore relies on the contract that the proto
// domain service implementation, when running inside a tx, returns
// the in-tx visible view of the schedules table. The adapter file
// documents the lock-key strategy.
//
// Sister agent contract (call sites)
// ----------------------------------
// `create.go` and `update.go` (sister-owned) MUST call:
//
//	if err := validateNoOverlap(ctx, uc.repositories.SupplierContractPriceSchedule,
//	    req.Data); err != nil { return nil, err }
//
// as the FIRST in-tx step before writing the candidate. The call
// either succeeds (signal: no overlap, OR open-ended truncate has
// been applied to the prior row), or returns an error explaining
// which existing row blocks the candidate.
package suppliercontractpriceschedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dbOps is the local capability surface validateNoOverlap needs from
// the database. The sister-agent's create.go / update.go pass the
// proto domain service directly — it satisfies dbOps because the
// signatures already match `SupplierContractPriceScheduleDomainServiceServer`.
//
// Declaring the narrow surface explicitly here lets the validator be
// unit-tested against a hand-rolled mock without depending on the
// full proto service.
type dbOps interface {
	ListSupplierContractPriceSchedules(
		ctx context.Context,
		in *scpspb.ListSupplierContractPriceSchedulesRequest,
	) (*scpspb.ListSupplierContractPriceSchedulesResponse, error)
	UpdateSupplierContractPriceSchedule(
		ctx context.Context,
		in *scpspb.UpdateSupplierContractPriceScheduleRequest,
	) (*scpspb.UpdateSupplierContractPriceScheduleResponse, error)
}

// validateNoOverlap inspects the candidate schedule against existing
// non-cancelled rows for the same contract.
//
// Behaviour:
//
//   - schedule slots into a free gap                      -> nil
//   - schedule overlaps an existing finite-end peer       -> error
//     naming the conflicting peer id and window
//   - schedule is open-ended AND a prior open-ended peer  -> the
//     prior peer is truncated (UPDATE issued through `ops`) so the
//     candidate INSERT will satisfy the partial unique index and
//     the GIST exclusion constraint; nil is returned to permit
//     the caller's INSERT to proceed
//   - schedule is itself CANCELLED                        -> error
//     (cancelled rows must use a different code path)
//   - candidate window is empty / inverted                -> error
//
// Required candidate fields: SupplierContractId, DateTimeStart,
// Status (must be != CANCELLED). Id may be empty for pre-INSERT
// validation; for UPDATE callers should populate Id so the row is
// excluded from its own comparison set.
//
// MUST be invoked inside a transaction the caller has opened.
func validateNoOverlap(
	ctx context.Context,
	ops dbOps,
	schedule *scpspb.SupplierContractPriceSchedule,
) error {
	if ops == nil {
		return errors.New("validateNoOverlap: dbOps is required")
	}
	if schedule == nil {
		return errors.New("validateNoOverlap: schedule is required")
	}
	if schedule.GetSupplierContractId() == "" {
		return errors.New("validateNoOverlap: schedule.supplier_contract_id is required")
	}
	if schedule.GetDateTimeStart() == nil {
		return errors.New("validateNoOverlap: schedule.date_time_start is required")
	}

	// Cancelled rows are not subject to overlap validation. They
	// have their own code path (cancel/supersede flows) that bypasses
	// this validator.
	if schedule.GetStatus() == scpspb.
		SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED {
		return errors.New("validateNoOverlap: cancelled schedules must not be validated for overlap")
	}

	// Normalise candidate window to UTC.
	candidateStart := schedule.GetDateTimeStart().AsTime().UTC()
	var candidateEnd time.Time
	candidateOpenEnded := schedule.GetDateTimeEnd() == nil
	if !candidateOpenEnded {
		candidateEnd = schedule.GetDateTimeEnd().AsTime().UTC()
		if !candidateEnd.After(candidateStart) {
			return fmt.Errorf(
				"validateNoOverlap: schedule period is empty or inverted: [%s, %s)",
				candidateStart.Format(time.RFC3339),
				candidateEnd.Format(time.RFC3339),
			)
		}
	}

	// Pull all schedules for the contract via the proto service.
	// Filtering by status would lose us the SUPERSEDED rows that
	// CRIT-2 requires we still consider, so we fetch the full set
	// and filter cancelled rows in-memory.
	supplierContractID := schedule.GetSupplierContractId()
	listReq := &scpspb.ListSupplierContractPriceSchedulesRequest{
		SupplierContractId: &supplierContractID,
	}
	if ws := schedule.GetWorkspaceId(); ws != "" {
		v := ws
		listReq.WorkspaceId = &v
	}
	listResp, err := ops.ListSupplierContractPriceSchedules(ctx, listReq)
	if err != nil {
		return fmt.Errorf("validateNoOverlap: list schedules for contract %s: %w",
			supplierContractID, err)
	}

	var openEndedPrior *scpspb.SupplierContractPriceSchedule
	for _, peer := range listResp.GetData() {
		if peer == nil {
			continue
		}
		if schedule.GetId() != "" && peer.GetId() == schedule.GetId() {
			continue // exclude self when validating an UPDATE
		}
		if peer.GetStatus() == scpspb.
			SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED {
			continue
		}
		if peer.GetDateTimeStart() == nil {
			continue
		}

		peerStart := peer.GetDateTimeStart().AsTime().UTC()
		peerOpenEnded := peer.GetDateTimeEnd() == nil
		var peerEnd time.Time
		if !peerOpenEnded {
			peerEnd = peer.GetDateTimeEnd().AsTime().UTC()
		}

		// Half-open overlap test: A=[aStart,aEnd) and B=[bStart,bEnd)
		// overlap iff aStart < bEnd AND bStart < aEnd. Open-ended
		// (no end) ranges treat the missing end as +Infinity.
		if !halfOpenOverlap(
			candidateStart, candidateEnd, candidateOpenEnded,
			peerStart, peerEnd, peerOpenEnded,
		) {
			continue
		}

		// CRIT-2 happy-path: candidate is the new open-ended row;
		// peer is the old open-ended row. Defer to truncate-and-
		// insert flow handled outside the loop.
		if candidateOpenEnded && peerOpenEnded {
			openEndedPrior = peer
			continue
		}

		// Any other overlap is a hard error.
		return fmt.Errorf(
			"schedule period %s overlaps existing schedule %s period %s",
			formatHalfOpen(candidateStart, candidateEnd, candidateOpenEnded),
			peer.GetId(),
			formatHalfOpen(peerStart, peerEnd, peerOpenEnded),
		)
	}

	if openEndedPrior == nil {
		return nil
	}

	// CRIT-2 truncate path. The candidate's start MUST be after the
	// prior row's start, otherwise we'd invert the prior window.
	priorStart := openEndedPrior.GetDateTimeStart().AsTime().UTC()
	if !candidateStart.After(priorStart) {
		return fmt.Errorf(
			"schedule start %s is not after the prior open-ended schedule %s start %s; cancel or supersede the prior row first",
			candidateStart.Format(time.RFC3339),
			openEndedPrior.GetId(),
			priorStart.Format(time.RFC3339),
		)
	}

	// Apply the in-tx UPDATE. proto.Clone is mandatory here — a
	// shallow `truncated := *openEndedPrior` would copy the proto's
	// internal `protoimpl.MessageState` (which embeds a sync.Mutex)
	// and trip `go vet`'s copylocks warning. The Update RPC returns
	// the canonical row; we discard the result because the caller
	// only needs to know the truncate succeeded.
	truncated := proto.Clone(openEndedPrior).(*scpspb.SupplierContractPriceSchedule)
	truncated.DateTimeEnd = timestamppbAt(candidateStart)
	now := time.Now().UTC()
	nowMillis := now.UnixMilli()
	nowStr := now.Format(time.RFC3339)
	truncated.DateModified = &nowMillis
	truncated.DateModifiedString = &nowStr

	if _, err := ops.UpdateSupplierContractPriceSchedule(ctx,
		&scpspb.UpdateSupplierContractPriceScheduleRequest{Data: truncated}); err != nil {
		return fmt.Errorf("validateNoOverlap: truncate prior open-ended schedule %s: %w",
			openEndedPrior.GetId(), err)
	}
	return nil
}

// halfOpenOverlap reports whether the two half-open windows overlap,
// treating an open-ended (no end) window as extending to +Infinity.
func halfOpenOverlap(
	aStart, aEnd time.Time, aOpen bool,
	bStart, bEnd time.Time, bOpen bool,
) bool {
	if !aOpen && !aEnd.After(bStart) {
		return false
	}
	if !bOpen && !bEnd.After(aStart) {
		return false
	}
	return true
}

// formatHalfOpen renders a window as "[start, end)" or "[start, ∞)"
// for inclusion in user-facing error messages. RFC3339 chosen for
// machine-friendly logs and unambiguous timezone display.
func formatHalfOpen(start, end time.Time, openEnded bool) string {
	if openEnded {
		return fmt.Sprintf("[%s, ∞)", start.Format(time.RFC3339))
	}
	return fmt.Sprintf("[%s, %s)", start.Format(time.RFC3339), end.Format(time.RFC3339))
}

// timestamppbAt is the local helper for the truncate UPDATE — wraps
// `timestamppb.New` with a clear name so the caller's intent is
// unambiguous next to the date_time_end field assignment.
func timestamppbAt(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }
