// Package procurementrequest contains the use cases for the
// ProcurementRequest entity.
//
// approve.go — CRIT-3 spawn-lifecycle saga.
//
// Why a saga (not a single big transaction)
// -----------------------------------------
// Approval transitions a PR through PENDING_APPROVAL ->
// APPROVED_PENDING_SPAWN -> (per-line spawn) -> APPROVED. The spawn
// step has to write into 3+ tables (procurement_request,
// procurement_request_line, plus one of supplier_contract /
// purchase_order_line_item / expenditure depending on
// `fulfillment_mode`) for every line. Plan §6.4b explicitly allows
// either pattern; this implementation chooses the SAGA variant for
// three reasons (codex-red-team CRIT-3 + plan rationale):
//
//  1. Cross-table atomicity is hostile under load. A single tx that
//     locks PR + N lines + N downstream spawn rows holds locks for the
//     duration of the slowest spawn. Even a small pool would serialise
//     approvals across an entire workspace.
//  2. AP-team retry is a real workflow. Operators need to retry only
//     the failed line of a 5-line PR — a single-tx implementation
//     forces "approve again from scratch" semantics, which collides
//     with already-spawned downstream artifacts.
//  3. Per-line `spawn_idempotency_key` is in proto for exactly this.
//     Each spawn helper is INSERT-ON-CONFLICT keyed and tolerates
//     re-invocation; the saga relies on that contract.
//
// Saga shape
// ----------
// Two phases run in distinct transactions:
//
//	PHASE 1 (atomic, small) — "claim approval intent"
//	  - Authcheck + validate inputs.
//	  - Read PR; assert status == PENDING_APPROVAL.
//	  - Run policy resolver per line (HIGH-5 fail-safe default).
//	  - Set PR.status = APPROVED_PENDING_SPAWN.
//	  - Set PR.approved_by, .approved_at.
//	  - For each line: spawn_status=PENDING, spawn_idempotency_key=
//	    "{request_id}:{line_id}:1".
//	  - Persist policy_decision_log JSON for audit.
//	  - Commit.
//
//	PHASE 2 (per-line, separate tx each) — "process spawn events"
//	  For each line in turn (in-process; the future outbox worker
//	  could parallelise this without changing the contract):
//	    - Begin tx.
//	    - Re-read line; if spawn_status==SPAWNED skip (idempotent).
//	    - Atomically claim: PENDING -> SPAWNING (CAS-style read +
//	      compare; double-claim returns and the worker logs a no-op).
//	    - Dispatch on fulfillment_mode -> spawn helper, which itself
//	      uses INSERT-ON-CONFLICT on the spawn_idempotency_key. The
//	      helper returns (spawned_id, error).
//	    - On success: set spawn_status=SPAWNED, populate the
//	      back-FK (spawned_supplier_contract_id /
//	      spawned_purchase_order_line_item_id /
//	      spawned_expenditure_id), set spawn_completed_at.
//	    - On error: set spawn_status=FAILED, populate spawn_error,
//	      set spawn_attempted_at. Move to next line.
//	    - Commit.
//
//	PHASE 3 (atomic, tiny) — "promote to APPROVED iff all lines done"
//	  - Begin tx.
//	  - Re-read all lines.
//	  - If every line.spawn_status == SPAWNED:
//	      PR.status = APPROVED. Commit.
//	  - Else:
//	      PR stays at APPROVED_PENDING_SPAWN with FAILED line(s)
//	      flagged (operator sees them in the detail page and can
//	      hit "Retry spawn" which calls this same code path with
//	      attempt=N+1 -> new idempotency key).
//
// Why phase 1 is its own tx (and not part of phase 2)
// ---------------------------------------------------
// The status transition to APPROVED_PENDING_SPAWN must be visible
// even if the very first spawn crashes the process. That's the
// contract that lets the "Retry spawn" UI work after a service
// restart — the operator sees a half-spawned PR, not a still-pending
// one. Phase 2 spawns are per-line tx for the same reason: line 4
// failing must not roll back lines 1–3.
//
// Idempotency contract with spawn helpers
// ---------------------------------------
// Each `Spawn*ForXxx` helper accepts the line + idempotency_key and
// is contractually required to:
//  1. INSERT...ON CONFLICT (idempotency_key) DO NOTHING and read back
//     the existing row when conflict.
//  2. Return the same id for the same key on retry.
//  3. Never partially write — either the downstream artifact exists
//     end-to-end or not at all.
//
// This use case never bypasses that contract. A new attempt is
// signalled by the caller bumping the suffix on
// `spawn_idempotency_key`; the helper then writes a NEW artifact and
// the previous failed one stays in place for audit (operator can
// cancel manually). The current implementation re-uses attempt=1 on
// first approval; the AP-team "Retry spawn" flow constructs the new
// idempotency key (out of scope for this file).
//
// HIGH-5 policy resolver plumbing
// -------------------------------
// `ApprovalPolicyResolver` is consulted at the START of phase 1 for
// every line BEFORE we transition the PR. The resolver is the single
// gate keeping PETTY auto-approve and FRAMEWORK skip-approval safe;
// implementations check regulatory class, supplier qualification,
// controlled-inventory tags, etc. The DEFAULT resolver
// (`DefaultRequireApprovalResolver`) returns RequiresApproval=true
// for every line — fail-safe. Workspaces register an override via
// `composition.RegisterApprovalPolicyResolver` (out of scope here).
//
// In this approve.go path, `RequiresApproval=true` is the EXPECTED
// outcome — we are inside the explicit operator-approval flow and
// the resolver's role is auditing (its `reason` is appended to
// `policy_decision_log`). The fast-path bypass is exercised only
// inside `submit.go`. We still honour the resolver on every line so
// the audit log is complete.
//
// Sister-agent ownership
// ----------------------
// The spawn helpers (`SpawnPurchaseOrderForOutright`,
// `SpawnPurchaseOrderForStockable`, `SpawnSupplierContractForRecurring`,
// `SpawnExpenditureForPetty`) and the policy registry plumbing live
// in sister-owned files. This file declares the interface boundary
// (`SpawnDispatcher`, `ApprovalPolicyResolver`) and consumes them.
package procurementrequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// SpawnDispatcher abstracts the four mode-specific spawn helpers
// behind a single dispatch surface. The sister agent owns the
// concrete implementation; this use case only depends on the
// behaviour contract documented at the top of this file (idempotent
// INSERT-ON-CONFLICT keyed on `idempotency_key`). The dispatcher is
// expected to populate the appropriate back-FK on the line and to
// return the downstream artifact id for logging.
type SpawnDispatcher interface {
	// SpawnForLine is invoked exactly once per (line, attempt). It
	// MUST:
	//   - INSERT-ON-CONFLICT on `idempotencyKey`.
	//   - Set the appropriate `spawned_*_id` back-FK on `line`
	//     (caller persists the line afterwards).
	//   - Return the spawned downstream artifact's id (for logging
	//     and operator surfacing) and the artifact-kind tag, OR an
	//     error to be recorded in `spawn_error`.
	SpawnForLine(
		ctx context.Context,
		line *procurementrequestlinepb.ProcurementRequestLine,
		idempotencyKey string,
	) (spawnedID string, artifactKind string, err error)
}

// ApprovalPolicyResolver — HIGH-5 hook (plan §6.4c). Checked here
// for audit-trail completeness even though approve.go is the
// require-approval path. Implementations are out of scope for SPS;
// the default resolver returns RequiresApproval=true for every line.
type ApprovalPolicyResolver interface {
	ResolveApprovalRequirement(
		ctx context.Context,
		line *procurementrequestlinepb.ProcurementRequestLine,
	) (requiresApproval bool, reason string, err error)
}

// DefaultRequireApprovalResolver is the fail-safe resolver: every
// line requires approval, reason "default policy". Wired by the
// container when no workspace-specific resolver is registered.
type DefaultRequireApprovalResolver struct{}

// ResolveApprovalRequirement returns RequiresApproval=true for every
// line.
func (DefaultRequireApprovalResolver) ResolveApprovalRequirement(
	_ context.Context,
	_ *procurementrequestlinepb.ProcurementRequestLine,
) (bool, string, error) {
	return true, "default policy: explicit approval required", nil
}

// ApproveProcurementRequestRepositories groups repository dependencies.
// Beyond the existing ProcurementRequest service we now also need
// the line service (to read & update lines) and the spawn dispatcher
// (to materialise downstream artifacts).
type ApproveProcurementRequestRepositories struct {
	ProcurementRequest     procurementrequestpb.ProcurementRequestDomainServiceServer
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
	SpawnDispatcher        SpawnDispatcher
}

// ApproveProcurementRequestServices groups service dependencies.
type ApproveProcurementRequestServices struct {
	AuthorizationService   ports.AuthorizationService
	TransactionService     ports.TransactionService
	TranslationService     ports.TranslationService
	IDService              ports.IDService
	ApprovalPolicyResolver ApprovalPolicyResolver // optional; nil falls back to DefaultRequireApprovalResolver
}

// ApproveProcurementRequestUseCase runs the CRIT-3 spawn-lifecycle
// saga.
type ApproveProcurementRequestUseCase struct {
	repositories ApproveProcurementRequestRepositories
	services     ApproveProcurementRequestServices
}

// NewApproveProcurementRequestUseCase wires dependencies.
func NewApproveProcurementRequestUseCase(
	repositories ApproveProcurementRequestRepositories,
	services ApproveProcurementRequestServices,
) *ApproveProcurementRequestUseCase {
	return &ApproveProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute runs the three-phase saga (claim -> per-line spawn ->
// promote). Returns the final ProcurementRequest in whichever status
// the saga ended at — APPROVED on full success, otherwise
// APPROVED_PENDING_SPAWN with one or more lines flagged FAILED.
func (uc *ApproveProcurementRequestUseCase) Execute(
	ctx context.Context,
	req *procurementrequestpb.ApproveProcurementRequestRequest,
) (*procurementrequestpb.ApproveProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.id_required",
			"Procurement request ID is required [DEFAULT]"))
	}
	if req.GetApprovedBy() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.approved_by_required",
			"Approver ID is required [DEFAULT]"))
	}
	if uc.repositories.SpawnDispatcher == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.errors.spawn_dispatcher_missing",
			"Spawn dispatcher is not configured [DEFAULT]"))
	}

	// PHASE 1 — claim approval intent in a small atomic tx.
	pr, lines, err := uc.claimApprovalIntent(ctx, req)
	if err != nil {
		return nil, err
	}

	// PHASE 2 — per-line spawn, each in its own tx so a single
	// failure leaves successful peers committed.
	for _, line := range lines {
		uc.processSpawnForLine(ctx, line)
	}

	// PHASE 3 — promote PR to APPROVED iff every line is SPAWNED.
	finalPR, err := uc.promoteIfAllSpawned(ctx, pr.GetId())
	if err != nil {
		return nil, err
	}

	return &procurementrequestpb.ApproveProcurementRequestResponse{
		Data:    finalPR,
		Success: true,
	}, nil
}

// claimApprovalIntent runs PHASE 1 in a single transaction.
func (uc *ApproveProcurementRequestUseCase) claimApprovalIntent(
	ctx context.Context,
	req *procurementrequestpb.ApproveProcurementRequestRequest,
) (*procurementrequestpb.ProcurementRequest, []*procurementrequestlinepb.ProcurementRequestLine, error) {
	var pr *procurementrequestpb.ProcurementRequest
	var lines []*procurementrequestlinepb.ProcurementRequestLine

	work := func(txCtx context.Context) error {
		readResp, err := uc.repositories.ProcurementRequest.ReadProcurementRequest(txCtx,
			&procurementrequestpb.ReadProcurementRequestRequest{
				Data: &procurementrequestpb.ProcurementRequest{Id: req.GetProcurementRequestId()},
			})
		if err != nil {
			return fmt.Errorf("read procurement request %s: %w", req.GetProcurementRequestId(), err)
		}
		prs := readResp.GetData()
		if len(prs) == 0 || prs[0] == nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService,
				"procurement_request.errors.not_found",
				"Procurement request not found [DEFAULT]"))
		}
		pr = prs[0]

		// Status guard: this use case only consumes PENDING_APPROVAL.
		// If the operator double-clicked, the second click sees
		// APPROVED_PENDING_SPAWN and we return idempotently —
		// phases 2 & 3 are themselves idempotent so re-running them
		// is safe and useful (it picks up any FAILED line for retry
		// when operator hits the button again).
		switch pr.GetStatus() {
		case procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_PENDING_APPROVAL:
			// fall through and claim
		case procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED_PENDING_SPAWN,
			procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED:
			// already claimed (or finalised) — load lines and let
			// phase 2 re-drive any FAILED ones.
			ln, err := uc.listLines(txCtx, pr.GetId())
			if err != nil {
				return err
			}
			lines = ln
			return nil
		default:
			return fmt.Errorf("procurement request %s is in status %s; only PENDING_APPROVAL can be approved",
				pr.GetId(), pr.GetStatus().String())
		}

		ln, err := uc.listLines(txCtx, pr.GetId())
		if err != nil {
			return err
		}
		if len(ln) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService,
				"procurement_request.errors.no_lines",
				"Procurement request has no lines to spawn [DEFAULT]"))
		}

		// Run the policy resolver for every line for audit. This is
		// the require-approval flow so RequiresApproval=true is
		// expected; we collect the reasons regardless.
		decisions, err := uc.collectPolicyDecisions(txCtx, ln)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		nowMillis := now.UnixMilli()
		approver := req.GetApprovedBy()
		approvedAtStr := now.Format(time.RFC3339)
		pendingSpawn := procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED_PENDING_SPAWN
		pr.Status = pendingSpawn
		pr.ApprovedBy = &approver
		pr.ApprovedAt = &nowMillis
		pr.ApprovedAtString = &approvedAtStr
		pr.DateModified = &nowMillis
		dms := approvedAtStr
		pr.DateModifiedString = &dms

		decisionsJSON, jerr := json.Marshal(decisions)
		if jerr == nil {
			s := string(decisionsJSON)
			pr.PolicyDecisionLog = &s
		}

		if _, err := uc.repositories.ProcurementRequest.UpdateProcurementRequest(txCtx,
			&procurementrequestpb.UpdateProcurementRequestRequest{Data: pr}); err != nil {
			return fmt.Errorf("update PR to APPROVED_PENDING_SPAWN: %w", err)
		}

		// Stamp each line PENDING with a fresh idempotency key. We
		// use attempt=1 here. Operator-driven retries (FAILED ->
		// PENDING) bump the suffix in a separate code path.
		for _, line := range ln {
			line.SpawnStatus = procurementrequestlinepb.
				ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_PENDING
			line.SpawnIdempotencyKey = deriveSpawnKey(pr.GetId(), line.GetId(), 1)
			line.SpawnError = nil
			line.SpawnAttemptedAt = nil
			line.SpawnCompletedAt = nil
			line.DateModified = &nowMillis
			lineDateModifiedStr := approvedAtStr
			line.DateModifiedString = &lineDateModifiedStr
			if _, err := uc.repositories.ProcurementRequestLine.UpdateProcurementRequestLine(txCtx,
				&procurementrequestlinepb.UpdateProcurementRequestLineRequest{Data: line}); err != nil {
				return fmt.Errorf("stamp line %s spawn metadata: %w", line.GetId(), err)
			}
		}
		lines = ln
		return nil
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, work); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
				"procurement_request.errors.approve_failed",
				"[ERR-DEFAULT] Failed to approve procurement request")
			return nil, nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := work(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
				"procurement_request.errors.approve_failed",
				"[ERR-DEFAULT] Failed to approve procurement request")
			return nil, nil, fmt.Errorf("%s: %w", translated, err)
		}
	}

	return pr, lines, nil
}

// processSpawnForLine runs PHASE 2 for a single line. Each call gets
// its own transaction so partial failure across N lines leaves N
// independent commits. Errors are recorded onto the line and never
// bubbled — phase 3 inspects the persisted spawn_status to decide
// promotion.
func (uc *ApproveProcurementRequestUseCase) processSpawnForLine(
	parentCtx context.Context,
	line *procurementrequestlinepb.ProcurementRequestLine,
) {
	work := func(txCtx context.Context) error {
		// Re-read the line under tx so we observe any concurrent
		// claim. This is the CAS step: PENDING -> SPAWNING. If
		// somebody beat us to it (SPAWNING / SPAWNED) we no-op.
		fresh, err := uc.readLine(txCtx, line.GetId())
		if err != nil {
			return err
		}
		switch fresh.GetSpawnStatus() {
		case procurementrequestlinepb.
			ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_SPAWNED:
			return nil // idempotent skip
		case procurementrequestlinepb.
			ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_SPAWNING:
			// Another worker holds the claim. We don't compete; the
			// holder will finish or mark FAILED. Return without
			// touching state.
			return nil
		}

		now := time.Now().UTC()
		nowMillis := now.UnixMilli()
		nowStr := now.Format(time.RFC3339)
		fresh.SpawnStatus = procurementrequestlinepb.
			ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_SPAWNING
		fresh.SpawnAttemptedAt = &nowMillis
		fresh.DateModified = &nowMillis
		fresh.DateModifiedString = &nowStr
		if _, err := uc.repositories.ProcurementRequestLine.UpdateProcurementRequestLine(txCtx,
			&procurementrequestlinepb.UpdateProcurementRequestLineRequest{Data: fresh}); err != nil {
			return fmt.Errorf("claim line %s -> SPAWNING: %w", fresh.GetId(), err)
		}

		// Dispatch to the mode-specific spawner. Helpers are
		// idempotent on idempotency_key (proto-level contract).
		spawnedID, kind, spawnErr := uc.repositories.SpawnDispatcher.SpawnForLine(
			txCtx, fresh, fresh.GetSpawnIdempotencyKey())
		_ = kind // logged via spawned_*_id back-FK below; kept for trace

		completedNow := time.Now().UTC()
		completedMillis := completedNow.UnixMilli()
		completedStr := completedNow.Format(time.RFC3339)
		fresh.DateModified = &completedMillis
		fresh.DateModifiedString = &completedStr

		if spawnErr != nil {
			// Mark FAILED but commit so operator sees the error.
			msg := spawnErr.Error()
			fresh.SpawnStatus = procurementrequestlinepb.
				ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_FAILED
			fresh.SpawnError = &msg
			if _, err := uc.repositories.ProcurementRequestLine.UpdateProcurementRequestLine(txCtx,
				&procurementrequestlinepb.UpdateProcurementRequestLineRequest{Data: fresh}); err != nil {
				return fmt.Errorf("record line %s spawn failure: %w", fresh.GetId(), err)
			}
			return nil
		}

		// Successful spawn — populate the back-FK appropriate to
		// the fulfillment_mode. The dispatcher is contracted to
		// have populated the back-FK on `fresh` already; the
		// fall-through here covers dispatchers that prefer the
		// caller to assign and is harmless when already set.
		if spawnedID != "" {
			switch fresh.GetFulfillmentMode() {
			case procurementrequestlinepb.
				ProcurementRequestLineFulfillmentMode_PROCUREMENT_REQUEST_LINE_FULFILLMENT_MODE_OUTRIGHT,
				procurementrequestlinepb.
					ProcurementRequestLineFulfillmentMode_PROCUREMENT_REQUEST_LINE_FULFILLMENT_MODE_STOCKABLE:
				if fresh.GetSpawnedPurchaseOrderLineItemId() == "" {
					id := spawnedID
					fresh.SpawnedPurchaseOrderLineItemId = &id
				}
			case procurementrequestlinepb.
				ProcurementRequestLineFulfillmentMode_PROCUREMENT_REQUEST_LINE_FULFILLMENT_MODE_RECURRING:
				if fresh.GetSpawnedSupplierContractId() == "" {
					id := spawnedID
					fresh.SpawnedSupplierContractId = &id
				}
			case procurementrequestlinepb.
				ProcurementRequestLineFulfillmentMode_PROCUREMENT_REQUEST_LINE_FULFILLMENT_MODE_PETTY:
				if fresh.GetSpawnedExpenditureId() == "" {
					id := spawnedID
					fresh.SpawnedExpenditureId = &id
				}
			}
		}
		fresh.SpawnStatus = procurementrequestlinepb.
			ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_SPAWNED
		fresh.SpawnError = nil
		fresh.SpawnCompletedAt = &completedMillis
		if _, err := uc.repositories.ProcurementRequestLine.UpdateProcurementRequestLine(txCtx,
			&procurementrequestlinepb.UpdateProcurementRequestLineRequest{Data: fresh}); err != nil {
			return fmt.Errorf("mark line %s SPAWNED: %w", fresh.GetId(), err)
		}
		return nil
	}

	// Per-line tx — failure inside `work` is logged but does not
	// bubble; the persisted spawn_status carries the outcome.
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		_ = uc.services.TransactionService.ExecuteInTransaction(parentCtx, work)
		return
	}
	_ = work(parentCtx)
}

// promoteIfAllSpawned runs PHASE 3. If every line for the PR is
// SPAWNED, the PR transitions APPROVED_PENDING_SPAWN -> APPROVED
// inside a small tx. Otherwise the PR is left at
// APPROVED_PENDING_SPAWN (the operator UI surfaces the failed lines).
func (uc *ApproveProcurementRequestUseCase) promoteIfAllSpawned(
	ctx context.Context,
	prID string,
) (*procurementrequestpb.ProcurementRequest, error) {
	var finalPR *procurementrequestpb.ProcurementRequest
	work := func(txCtx context.Context) error {
		readResp, err := uc.repositories.ProcurementRequest.ReadProcurementRequest(txCtx,
			&procurementrequestpb.ReadProcurementRequestRequest{
				Data: &procurementrequestpb.ProcurementRequest{Id: prID},
			})
		if err != nil {
			return fmt.Errorf("re-read PR %s for promotion: %w", prID, err)
		}
		prs := readResp.GetData()
		if len(prs) == 0 || prs[0] == nil {
			return fmt.Errorf("PR %s vanished mid-saga", prID)
		}
		pr := prs[0]
		if pr.GetStatus() == procurementrequestpb.
			ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED {
			finalPR = pr
			return nil
		}

		lines, err := uc.listLines(txCtx, prID)
		if err != nil {
			return err
		}
		allSpawned := len(lines) > 0
		for _, line := range lines {
			if line.GetSpawnStatus() != procurementrequestlinepb.
				ProcurementRequestLineSpawnStatus_PROCUREMENT_REQUEST_LINE_SPAWN_STATUS_SPAWNED {
				allSpawned = false
				break
			}
		}
		if !allSpawned {
			finalPR = pr
			return nil
		}

		now := time.Now().UTC()
		nowMillis := now.UnixMilli()
		nowStr := now.Format(time.RFC3339)
		pr.Status = procurementrequestpb.
			ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED
		pr.DateModified = &nowMillis
		pr.DateModifiedString = &nowStr
		updateResp, err := uc.repositories.ProcurementRequest.UpdateProcurementRequest(txCtx,
			&procurementrequestpb.UpdateProcurementRequestRequest{Data: pr})
		if err != nil {
			return fmt.Errorf("promote PR %s to APPROVED: %w", prID, err)
		}
		if data := updateResp.GetData(); len(data) > 0 && data[0] != nil {
			finalPR = data[0]
		} else {
			finalPR = pr
		}
		return nil
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, work); err != nil {
			return nil, err
		}
	} else if err := work(ctx); err != nil {
		return nil, err
	}
	return finalPR, nil
}

// listLines fetches all lines for the PR. Implemented as a small
// helper because both phase 1 and phase 3 need the same query and
// they each run inside their own tx context.
func (uc *ApproveProcurementRequestUseCase) listLines(
	ctx context.Context,
	prID string,
) ([]*procurementrequestlinepb.ProcurementRequestLine, error) {
	id := prID
	resp, err := uc.repositories.ProcurementRequestLine.ListProcurementRequestLines(ctx,
		&procurementrequestlinepb.ListProcurementRequestLinesRequest{
			ProcurementRequestId: &id,
		})
	if err != nil {
		return nil, fmt.Errorf("list lines for PR %s: %w", prID, err)
	}
	return resp.GetData(), nil
}

// readLine fetches a single line by id under the current ctx (which
// the caller has typically wrapped in a transaction).
func (uc *ApproveProcurementRequestUseCase) readLine(
	ctx context.Context,
	lineID string,
) (*procurementrequestlinepb.ProcurementRequestLine, error) {
	resp, err := uc.repositories.ProcurementRequestLine.ReadProcurementRequestLine(ctx,
		&procurementrequestlinepb.ReadProcurementRequestLineRequest{
			Data: &procurementrequestlinepb.ProcurementRequestLine{Id: lineID},
		})
	if err != nil {
		return nil, fmt.Errorf("read line %s: %w", lineID, err)
	}
	rows := resp.GetData()
	if len(rows) == 0 || rows[0] == nil {
		return nil, fmt.Errorf("line %s not found", lineID)
	}
	return rows[0], nil
}

// policyDecision is the in-memory shape we write to
// ProcurementRequest.policy_decision_log as a JSON array — one entry
// per line per resolver invocation. Auditors read this column to
// answer "why did line X get approved without explicit review?".
// Even on the require-approval path the log is useful: it captures
// resolver opinion at approval time so a later policy change is
// distinguishable from a historical decision.
type policyDecision struct {
	LineID           string `json:"line_id"`
	RequiresApproval bool   `json:"requires_approval"`
	Reason           string `json:"reason"`
	ResolvedAt       string `json:"resolved_at"`
}

// collectPolicyDecisions runs the resolver for every line and
// returns the audit array. Falls back to DefaultRequireApprovalResolver
// when no resolver is wired (fail-safe per HIGH-5).
func (uc *ApproveProcurementRequestUseCase) collectPolicyDecisions(
	ctx context.Context,
	lines []*procurementrequestlinepb.ProcurementRequestLine,
) ([]policyDecision, error) {
	resolver := uc.services.ApprovalPolicyResolver
	if resolver == nil {
		resolver = DefaultRequireApprovalResolver{}
	}
	out := make([]policyDecision, 0, len(lines))
	for _, line := range lines {
		req, reason, err := resolver.ResolveApprovalRequirement(ctx, line)
		if err != nil {
			return nil, fmt.Errorf("policy resolver for line %s: %w", line.GetId(), err)
		}
		out = append(out, policyDecision{
			LineID:           line.GetId(),
			RequiresApproval: req,
			Reason:           reason,
			ResolvedAt:       time.Now().UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

// deriveSpawnKey constructs the deterministic idempotency key used
// by spawn helpers. Format documented in proto:
//
//	"{request_id}:{line_id}:{attempt_n}"
//
// Operator-driven retries bump `attempt_n` to force a new downstream
// artifact while preserving the audit trail of the previous failed
// one. Centralised here so all callers (initial approve + future
// retry use case) agree on the format.
func deriveSpawnKey(requestID, lineID string, attempt int) string {
	return fmt.Sprintf("%s:%s:%d", requestID, lineID, attempt)
}
