package grade_compute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
)

// recompute_intx.go — the submit-time freshness barrier seam (FIX-3 / codex-
// rereview.md "Exact lock and state contract" step 4). The job_phase submit
// transition, holding the parent mutex over the locked sheet, calls SheetRecompute
// (injected as a port in operation/usecases.go) to finalize phase then job outcome
// summaries INSIDE the transition transaction BEFORE flipping IN_PROGRESS →
// FOR_REVIEW. A genuine recompute failure rolls the whole transition back; a phase
// that is simply not gradable, or gradable-but-blank, is an expected skip (D6
// permits partial/blank submission).
//
// These entry points run the roll-up CORE directly on the ambient ctx executor,
// deliberately bypassing:
//   - the PhaseOutcomeSummary/JobOutcomeSummary :create ACTION GATE — the caller (a
//     job_phase transition) has already strictly authorized the action; a teacher or
//     reviewer legitimately holds no summary-create grant, so re-gating here would
//     wrongly deny the system finalization; and
//   - the use case's own ExecuteInTransaction wrapper — the Transactor is re-entrant
//     (adapter/core/transactions.go RunInTransactionWithOptions joins an active tx),
//     so Execute would join the ambient tx anyway; calling the core avoids the
//     redundant gate while still running every write on GetExecutor(ctx) = the
//     submit's *sql.Tx.

// RecomputePhaseInAmbientTx finalizes ONE phase's phase_outcome_summary inside the
// caller's already-open transaction. Returns (recomputed, err):
//   - (true,  nil)  a summary was (re)written for a gradable phase with data;
//   - (false, nil)  an EXPECTED skip — the phase is not gradable (no scheme / no
//     score scale / no scoped criteria) OR is gradable but has no recorded values
//     yet (a blank sheet under D6);
//   - (_,     err)  a genuine read/config failure that MUST roll the transition back.
func (uc *ComputePhaseOutcomeUseCase) RecomputePhaseInAmbientTx(ctx context.Context, jobPhaseID string) (bool, error) {
	// The eligibility classifier reuses the roll-up's own scheme-resolution ladder,
	// so "not gradable" here can never disagree with what executeCore would attempt.
	eligible, _, err := uc.CheckRecomputeEligibility(ctx, jobPhaseID)
	if err != nil {
		return false, err
	}
	if !eligible {
		return false, nil // ledger/unschemed phase → nothing to finalize
	}
	if _, err := uc.executeCore(ctx, &ComputePhaseOutcomeRequest{JobPhaseId: jobPhaseID}); err != nil {
		if errors.Is(err, ErrNoRecordedValues) {
			return false, nil // gradable but blank — expected skip
		}
		return false, err
	}
	return true, nil
}

// RecomputeJobInAmbientTx finalizes ONE job's year-final job_outcome_summary +
// job_outcome_line inside the caller's transaction (same gate/tx-skip rationale as
// RecomputePhaseInAmbientTx). Returns (false,nil) when the job has no graded phases
// yet (all-blank) or its summary is authoritative/frozen (ErrSummaryFrozen) — both
// expected skips; (_, err) only on a genuine failure.
func (uc *ComputeJobOutcomeUseCase) RecomputeJobInAmbientTx(ctx context.Context, jobID string) (bool, error) {
	if _, err := uc.executeJobCore(ctx, &ComputeJobOutcomeRequest{JobId: jobID}); err != nil {
		if errors.Is(err, ErrSummaryFrozen) || errors.Is(err, ErrNoGradedPhases) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// phaseRecomputer / jobRecomputer are the narrow seams recomputeSheet depends on,
// satisfied by *ComputePhaseOutcomeUseCase / *ComputeJobOutcomeUseCase. They exist
// so the sheet-recompute orchestration (skip-vs-fail propagation) is unit-testable
// without a database.
type phaseRecomputer interface {
	RecomputePhaseInAmbientTx(ctx context.Context, jobPhaseID string) (bool, error)
}
type jobRecomputer interface {
	RecomputeJobInAmbientTx(ctx context.Context, jobID string) (bool, error)
}

// recomputeSheet finalizes the phase summaries then the job summaries for a locked
// sheet, propagating the FIRST genuine failure (which rolls the transition tx back).
// Expected skips (false,nil) are ignored. Phase-before-job ordering matches the
// global lock order (phase_outcome_summary before job_outcome_summary) and the
// year-final pass-through dependency (the job roll-up reads the phase summaries this
// pass just wrote).
func recomputeSheet(ctx context.Context, phase phaseRecomputer, job jobRecomputer, phaseIDs, jobIDs []string) error {
	for _, pid := range phaseIDs {
		if pid == "" {
			continue
		}
		if _, err := phase.RecomputePhaseInAmbientTx(ctx, pid); err != nil {
			return fmt.Errorf("sheet recompute: phase %s: %w", pid, err)
		}
	}
	for _, jid := range jobIDs {
		if jid == "" {
			continue
		}
		if _, err := job.RecomputeJobInAmbientTx(ctx, jid); err != nil {
			return fmt.Errorf("sheet recompute: job %s: %w", jid, err)
		}
	}
	return nil
}

// SheetRecompute is the injected freshness-barrier port the job_phase submit
// transition calls post-lock/pre-flip (wired in operation/usecases.go, type-asserted
// onto the job_phase adapter). It runs on the ambient transition transaction; a
// non-nil return rolls the transition back. Its signature is exactly
// func(context.Context, []string, []string) error — the bare type the adapter's
// SetSheetRecompute setter accepts, so no shared named type is needed.
func (uc *UseCases) SheetRecompute(ctx context.Context, phaseIDs, jobIDs []string) error {
	if uc == nil || uc.ComputePhaseOutcome == nil || uc.ComputeJobOutcome == nil {
		return fmt.Errorf("sheet recompute: grade-compute use cases are not wired")
	}
	// Mark the whole recompute subtree as the trusted system seam (codex P3 §A3):
	// every summary/outcome read beneath this point reads on the AMBIENT transition
	// executor and drops the STAFF row scope, so the job roll-up sees the phase
	// summaries this pass writes in the SAME tx and a teacher-scoped submit
	// recomputes over the full sheet's inputs — not a partial, pool-visible subset.
	ctx = approvalctx.WithTrustedRecompute(ctx)
	return recomputeSheet(ctx, uc.ComputePhaseOutcome, uc.ComputeJobOutcome, phaseIDs, jobIDs)
}
