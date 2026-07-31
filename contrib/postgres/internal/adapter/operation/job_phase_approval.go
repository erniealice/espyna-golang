//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/registry/entityid"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// ============================================================================
// Per-phase approval transitions — the binding §4.2 contract
// (codex-rereview.md "Exact lock and state contract" + "Exact partial-
// reachability result" + "Exact E1 mixed-state recovery" + "Audit snapshot
// and retention contract").
//
// Every transition runs in ONE transaction with the ambient executor
// (dbOps.GetExecutor(ctx)); it FAILS CLOSED when no transaction is active — an
// out-of-transaction FOR UPDATE releases its locks immediately and provides no
// serialization, so a non-transactional path is never a safe fallback here.
//
// Global lock order (all approval-aware paths): job_template_phase parent →
// job_phase asc id → (task/task_outcome) → phase_outcome_summary →
// job_outcome_summary. Transitions take the ancestry-validated parent FOR
// UPDATE (the parent mutex) and lock the full sheet with `ORDER BY jp.id FOR
// UPDATE OF jp` so joined `job` rows are read but not row-marked.
// ============================================================================

// Persisted approval enum tokens (the DB stores the enum NAME as TEXT; UNSPECIFIED
// never persists — DB CHECK admits only these four).
const (
	apInProgress = "PHASE_APPROVAL_STATUS_IN_PROGRESS"
	apForReview  = "PHASE_APPROVAL_STATUS_FOR_REVIEW"
	apVerified   = "PHASE_APPROVAL_STATUS_VERIFIED"
	apPublished  = "PHASE_APPROVAL_STATUS_PUBLISHED"
)

const returnReasonMaxLen = 2000

// jobPhaseParentLockSQL is the parent mutex: prove jtp.id=$1(phase) AND
// jtp.job_template_id=$2(template) AND the owning job_template is in the trusted
// workspace ($3) OR global (workspace_id IS NULL), then lock the parent row
// (codex §1 HIGH: the parent-lock statement must itself carry the trusted
// workspace/global ancestry — proving tenant binding only on the later child
// query lets a cross-tenant caller take the parent mutex first). job_template_phase
// itself is template/global-scoped (no workspace_id), so the bind rides on the
// joined job_template. FOR UPDATE OF jtp row-marks ONLY the parent phase — the
// joined job_template is read, not locked, so a global template shared across
// workspaces is not serialized tenant-against-tenant on its own row.
const jobPhaseParentLockSQL = `
	SELECT jtp.id
	FROM ` + entityid.JobTemplatePhase + ` jtp
	JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
	WHERE jtp.id = $1 AND jtp.job_template_id = $2 AND jtp.active = true
	  AND (jt.workspace_id = $3 OR jt.workspace_id IS NULL)
	FOR UPDATE OF jtp`

// jobPhaseSheetLockSQL locks the FULL sheet set S — NO staff predicate. It proves
// jp.template_phase_id=jtp.id ($2=phase) AND j.job_template_id=jtp.job_template_id
// ($1=template) AND binds j.workspace_id ($3, trusted). FOR UPDATE OF jp row-marks
// ONLY job_phase; the joined job rows are read, not locked (so two transitions
// over different phases cannot form a lock-ordering cycle on shared job rows).
func jobPhaseSheetLockSQL(groupNarrow string) string {
	return `
	SELECT jp.id, jp.job_id, jp.approval_status
	FROM ` + entityid.JobPhase + ` jp
	JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
	WHERE jp.template_phase_id = $2
	  AND j.job_template_id = $1
	  AND j.workspace_id = $3
	  AND jp.active = true` + groupNarrow + `
	ORDER BY jp.id
	FOR UPDATE OF jp`
}

// normalizeReturnReason trims, caps at returnReasonMaxLen, and enforces the
// published-return reason requirement. published==true requires a non-blank
// reason (codex "Published-return and lock overlay contract").
func normalizeReturnReason(raw string, published bool) (string, error) {
	reason := strings.TrimSpace(raw)
	if len(reason) > returnReasonMaxLen {
		reason = reason[:returnReasonMaxLen]
	}
	if published && reason == "" {
		return "", fmt.Errorf("job_phase return: a return involving a published phase requires a non-blank reason")
	}
	return reason, nil
}

// approvalExecutorProvider is the narrow type-assertion used to obtain the
// transaction-aware executor (mirrors the seat adapter's seatExecutorProvider).
type approvalExecutorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// txExecutor returns the ambient *sql.Tx executor, or a fail-closed error when
// no transaction is active. This is the "NO non-transactional fallback" gate:
// GetExecutor returns *sql.Tx only inside a pending transaction, otherwise the
// pool *sql.DB — and a *sql.DB FOR UPDATE would not serialize.
func (r *PostgresJobPhaseRepository) txExecutor(ctx context.Context) (sqlexec.DBExecutor, error) {
	ep, ok := r.dbOps.(approvalExecutorProvider)
	if !ok {
		return nil, fmt.Errorf("job_phase approval: dbOps does not provide a transaction-aware executor")
	}
	exec := ep.GetExecutor(ctx)
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("job_phase approval: transition requires an ambient transaction (fail-closed)")
	}
	return exec, nil
}

// readExecutor returns the ambient executor (tx if active, else pool) for reads
// that do not themselves require a transaction.
func (r *PostgresJobPhaseRepository) readExecutor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(approvalExecutorProvider); ok {
		return ep.GetExecutor(ctx)
	}
	return r.db
}

// dropStaffScopeForRecompute reports whether the per-request STAFF row scope must
// be dropped for the trusted submit-time freshness-barrier recompute seam
// (approvalctx.IsTrustedRecompute — set ONLY by grade_compute.SheetRecompute
// inside the transition tx). It fails closed unless an ambient *sql.Tx is active,
// so the marker can NEVER become a pool-based staff-scope bypass: the trusted
// unscoped read is only ever served on the transition's own transaction executor.
// The summary/outcome reads consult this so the system finalization (a) sees the
// full sheet's inputs regardless of the acting teacher's row scope and (b) sees
// its own in-tx summary writes (codex P3 §A3).
func dropStaffScopeForRecompute(ctx context.Context, exec sqlexec.DBExecutor) (bool, error) {
	if !approvalctx.IsTrustedRecompute(ctx) {
		return false, nil
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return false, fmt.Errorf("trusted recompute read: requires an ambient transaction (fail closed)")
	}
	return true, nil
}

// LockTemplatePhasesForSpawn is the W-SPAWN parent mutex: it pre-locks the given
// job_template_phase parents FOR UPDATE in sorted id order (the same parents +
// same global lock order the transitions take) INSIDE the ambient transaction. It
// FAILS CLOSED when no transaction is active — that is how "phase creation
// requires the ambient transaction" is enforced for the postgres path. Idempotent
// on an empty id list.
func (r *PostgresJobPhaseRepository) LockTemplatePhasesForSpawn(ctx context.Context, templatePhaseIDs []string) error {
	if len(templatePhaseIDs) == 0 {
		return nil
	}
	exec, err := r.txExecutor(ctx)
	if err != nil {
		return err
	}
	ids := append([]string(nil), templatePhaseIDs...)
	sort.Strings(ids)
	// ORDER BY id before FOR UPDATE so rows are lock-acquired in a single global
	// order (avoids a lock-ordering cycle with a concurrent spawn/transition).
	const q = `SELECT id FROM ` + entityid.JobTemplatePhase + ` WHERE id = ANY($1) ORDER BY id FOR UPDATE`
	rows, err := exec.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("job_phase spawn: lock template phases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("job_phase spawn: scan locked template phase: %w", err)
		}
	}
	return rows.Err()
}

// JobHardFrozen reports whether ONE job is hard-frozen (active authoritative
// final OR closed enclosing academic-year price_schedule) — the same predicate
// sheetHardFrozen applies per job. Used to reject a late W-SPAWN into finalized
// history. The job is bound to the trusted workspace ($2) and every freeze-source
// relation (subscription_group_member/group/schedule) is required to share the
// job's workspace, so an independently writable cross-workspace group/schedule
// cannot freeze another tenant's job (codex §6 HIGH: JobHardFrozen had no
// workspace predicate at all). An empty wsID matches no row (fail-closed bind).
// job_outcome_summary IS workspace-bearing (job_outcome_summary.proto:53
// workspace_id = 28), so the authoritative-summary EXISTS clause additionally
// requires jos.workspace_id = j.workspace_id (codex P3 §A6: a foreign-tenant
// authoritative summary carrying tenant A's job_id must NOT freeze A's job).
func (r *PostgresJobPhaseRepository) JobHardFrozen(ctx context.Context, jobID, wsID string) (bool, error) {
	return jobHardFrozen(ctx, r.readExecutor(ctx), jobID, wsID)
}

// jobHardFrozen is the reusable per-job freeze predicate (shared by the method,
// the cell-write guard, and any other approval-aware path). Every freeze-source
// relation is workspace-bound to the job (codex §6 HIGH + P3 §A6).
func jobHardFrozen(ctx context.Context, exec sqlexec.DBExecutor, jobID, wsID string) (bool, error) {
	if jobID == "" {
		return false, nil
	}
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM ` + entityid.Job + ` j
			WHERE j.id = $1 AND j.workspace_id = $2 AND (
				EXISTS (
					SELECT 1 FROM ` + entityid.JobOutcomeSummary + ` jos
					WHERE jos.job_id = j.id AND jos.workspace_id = j.workspace_id
					  AND jos.active = true AND jos.is_authoritative = true
				)
				OR EXISTS (
					SELECT 1 FROM ` + entityid.SubscriptionGroupMember + ` sgm
					JOIN ` + entityid.SubscriptionGroup + ` sg ON sg.id = sgm.subscription_group_id
					JOIN ` + entityid.PriceSchedule + ` ps ON ps.id = sg.price_schedule_id
					WHERE j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
					  AND sgm.subscription_id = j.origin_id
					  AND sgm.workspace_id = j.workspace_id
					  AND sg.workspace_id = j.workspace_id
					  AND ps.workspace_id = j.workspace_id
					  AND ps.closed = true
				)
			)
		)`
	var frozen bool
	if err := exec.QueryRowContext(ctx, q, jobID, wsID).Scan(&frozen); err != nil {
		return false, fmt.Errorf("job_phase: hard_frozen probe (job=%s): %w", jobID, err)
	}
	return frozen, nil
}

// guardCellWrite implements the ordinary CELL-WRITE lock protocol (plan §4.2
// "Cell writes … parent FOR SHARE → job_phase row lock → recheck …" and codex
// §1/§2 CRITICAL). Every production task_outcome / job_task mutation that touches
// a matrix cell MUST call this INSIDE the ambient transaction before writing the
// leaf, so a concurrent approval transition cannot flip the phase out from under
// an in-flight edit (and vice versa). Given a job_task it:
//  1. resolves the job_task → job_phase → job ancestry under the trusted
//     workspace (an unlocked read; membership anchors are now immutable so the
//     resolved parent is stable);
//  2. locks the parent job_template_phase FOR SHARE (global lock order step 1 —
//     incompatible with a transition's parent FOR UPDATE, so the two serialize);
//  3. locks the owning job_phase row FOR UPDATE and RE-READS its approval status
//     under the lock;
//  4. rechecks the status is IN_PROGRESS (FOR_REVIEW/VERIFIED/PUBLISHED are
//     workflow-locked — the cell is not editable);
//  5. rechecks the job is not hard-frozen (closed schedule / authoritative final).
//
// Fails closed on any missing ancestry, wrong workspace, advanced status, or
// freeze. exec MUST be the ambient *sql.Tx; an out-of-transaction lock releases
// immediately and provides no serialization.
func guardCellWrite(ctx context.Context, exec sqlexec.DBExecutor, jobTaskID, wsID string) error {
	if jobTaskID == "" {
		return fmt.Errorf("cell write guard: job_task_id is required (fail closed)")
	}
	if wsID == "" {
		return fmt.Errorf("cell write guard: no trusted workspace (fail closed)")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("cell write guard: requires an ambient transaction (fail closed)")
	}

	var jobPhaseID, templatePhaseID, jobID string
	const resolveSQL = `
		SELECT jp.id, COALESCE(jp.template_phase_id, ''), j.id
		FROM ` + entityid.JobTask + ` jt
		JOIN ` + entityid.JobPhase + ` jp ON jp.id = jt.job_phase_id AND jp.active = true
		JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
		WHERE jt.id = $1 AND jt.active = true AND j.workspace_id = $2`
	if err := exec.QueryRowContext(ctx, resolveSQL, jobTaskID, wsID).Scan(&jobPhaseID, &templatePhaseID, &jobID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("cell write guard: job_task not found / not in workspace / inactive ancestry — fail closed")
		}
		return fmt.Errorf("cell write guard: resolve ancestry: %w", err)
	}

	// 2. Parent mutex (SHARE) — serializes against a transition's parent FOR UPDATE.
	if templatePhaseID != "" {
		if _, err := exec.ExecContext(ctx,
			`SELECT id FROM `+entityid.JobTemplatePhase+` WHERE id = $1 FOR SHARE`, templatePhaseID); err != nil {
			return fmt.Errorf("cell write guard: lock parent FOR SHARE: %w", err)
		}
	}

	// 3. Lock the owning job_phase row and re-read status under the lock.
	var status string
	if err := exec.QueryRowContext(ctx,
		`SELECT approval_status FROM `+entityid.JobPhase+` WHERE id = $1 AND active = true FOR UPDATE`,
		jobPhaseID).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("cell write guard: job_phase vanished under lock — fail closed")
		}
		return fmt.Errorf("cell write guard: lock job_phase FOR UPDATE: %w", err)
	}

	// 3b. Recheck the membership anchor UNDER the locks (codex P3 §A2): lock the
	// job_task row FOR UPDATE (global lock order: parent → job_phase → task) and
	// re-read its job_phase_id/active. The step-1 resolve was an unlocked read; this
	// re-verifies the task still belongs to THIS locked phase and is still active, so
	// even though job_task.job_phase_id is now enforced immutable, a concurrent
	// deactivate/repoint cannot slip a write past the guard.
	var stillPhase string
	var stillActive bool
	if err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(job_phase_id, ''), active FROM `+entityid.JobTask+` WHERE id = $1 FOR UPDATE`,
		jobTaskID).Scan(&stillPhase, &stillActive); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("cell write guard: job_task vanished under lock — fail closed")
		}
		return fmt.Errorf("cell write guard: lock job_task FOR UPDATE: %w", err)
	}
	if !stillActive || stillPhase != jobPhaseID {
		return fmt.Errorf("cell write guard: job_task membership changed under lock (phase=%s active=%t) — fail closed", stillPhase, stillActive)
	}

	// 4. Only IN_PROGRESS phases are editable.
	if status != apInProgress {
		return fmt.Errorf("cell write guard: phase %s is %s (workflow-locked) — cell not editable, fail closed", jobPhaseID, status)
	}

	// 5. Hard-frozen recheck (closed schedule / authoritative final), workspace-scoped.
	frozen, err := jobHardFrozen(ctx, exec, jobID, wsID)
	if err != nil {
		return err
	}
	if frozen {
		return fmt.Errorf("cell write guard: job %s is hard-frozen — cell not editable, fail closed", jobID)
	}
	return nil
}

// guardPhaseWrite is the CELL-WRITE lock protocol keyed by a TARGET job_phase
// rather than an existing job_task — used when CREATING a new job_task under a
// phase (the task does not exist yet, so guardCellWrite's job_task resolve cannot
// run). It resolves job_phase → job under the trusted workspace, takes the parent
// job_template_phase FOR SHARE + the job_phase row FOR UPDATE, and rechecks
// IN_PROGRESS + not hard-frozen (codex P3 §A2: generic JobTask create must be
// guarded — adding a task to an advanced/frozen sheet is a membership/data-grain
// mutation). Fails closed on missing ancestry, wrong workspace, advanced status,
// freeze, or a non-tx executor.
func guardPhaseWrite(ctx context.Context, exec sqlexec.DBExecutor, jobPhaseID, wsID string) error {
	if jobPhaseID == "" {
		return fmt.Errorf("phase write guard: job_phase_id is required (fail closed)")
	}
	if wsID == "" {
		return fmt.Errorf("phase write guard: no trusted workspace (fail closed)")
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("phase write guard: requires an ambient transaction (fail closed)")
	}

	var templatePhaseID, jobID string
	const resolveSQL = `
		SELECT COALESCE(jp.template_phase_id, ''), j.id
		FROM ` + entityid.JobPhase + ` jp
		JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
		WHERE jp.id = $1 AND jp.active = true AND j.workspace_id = $2`
	if err := exec.QueryRowContext(ctx, resolveSQL, jobPhaseID, wsID).Scan(&templatePhaseID, &jobID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("phase write guard: job_phase not found / not in workspace / inactive — fail closed")
		}
		return fmt.Errorf("phase write guard: resolve ancestry: %w", err)
	}

	if templatePhaseID != "" {
		if _, err := exec.ExecContext(ctx,
			`SELECT id FROM `+entityid.JobTemplatePhase+` WHERE id = $1 FOR SHARE`, templatePhaseID); err != nil {
			return fmt.Errorf("phase write guard: lock parent FOR SHARE: %w", err)
		}
	}

	var status string
	if err := exec.QueryRowContext(ctx,
		`SELECT approval_status FROM `+entityid.JobPhase+` WHERE id = $1 AND active = true FOR UPDATE`,
		jobPhaseID).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("phase write guard: job_phase vanished under lock — fail closed")
		}
		return fmt.Errorf("phase write guard: lock job_phase FOR UPDATE: %w", err)
	}
	if status != apInProgress {
		return fmt.Errorf("phase write guard: phase %s is %s (workflow-locked) — cannot add a task, fail closed", jobPhaseID, status)
	}

	frozen, err := jobHardFrozen(ctx, exec, jobID, wsID)
	if err != nil {
		return err
	}
	if frozen {
		return fmt.Errorf("phase write guard: job %s is hard-frozen — cannot add a task, fail closed", jobID)
	}
	return nil
}

// lockedPhase is one row of the locked sheet set S.
type lockedPhase struct {
	id             string
	jobID          string
	approvalStatus string
}

// lockParentAndSheet proves parent ancestry, takes the parent job_template_phase
// FOR UPDATE (the parent mutex), then locks the full nonempty sheet set S with
// `ORDER BY jp.id FOR UPDATE OF jp` under a trusted workspace bind. Returns the
// locked rows in ascending id order. An empty S is a typed error (nothing to
// transition / IDOR mismatch is indistinguishable from an empty sheet).

// groupNarrowPredicate returns the SQL fragment restricting a sheet set S to ONE
// delivery group, plus the arg to append. Empty groupID returns ("", nil), so a
// transition WITHOUT a group is byte-identical to its pre-20260725 form — the
// whole feature is gated on the caller actually passing one.
//
// THE JOIN IS DELIBERATELY TIGHT: membership AND the job's originating
// subscription (sgm_g.subscription_id = j.origin_id), matching the delivery
// aggregate's own link — NOT bare membership. Bare membership is wrong here in a
// way that is easy to miss: a student holds ACTIVE membership in a group for
// every academic year they were enrolled, so a group from another year would
// select a plausible SUBSET of this template's jobs (measured on education1:
// 2,440 such (template, group) pairs, worst case 27 students). For a read that
// is a wrong roster; for a WRITE it would transition the wrong students'
// approval state. Pinning origin_id selects exactly the jobs that group's
// subscription produced.
//
// The membership row is bound to the caller's workspace placeholder (wsArgN), so
// a foreign-workspace group selects nothing rather than erroring, and
// lockParentAndSheet's existing empty-set guard turns that into a fail-closed
// "empty target" error. Both placeholder indexes are caller-supplied because the
// surrounding queries number their args differently (the transition probes bind
// workspace at $3, the read roll-up at $2).
//
// Shared by the WRITE path (transitions, this file) and the READ path
// (outcome_matrix_query.go's approval roll-up) on purpose: if the two ever
// disagreed about which jobs belong to a group, a sheet would display one set
// and transition another.
func groupNarrowPredicate(groupID string, groupArgN, wsArgN int) (string, []any) {
	if groupID == "" {
		return "", nil
	}
	return fmt.Sprintf(`
			  AND EXISTS (
			    SELECT 1 FROM `+entityid.SubscriptionGroupMember+` sgm_g
			    WHERE sgm_g.client_id = j.client_id
			      AND sgm_g.subscription_id = j.origin_id
			      AND sgm_g.subscription_group_id = $%d
			      AND sgm_g.workspace_id = $%d
			      AND sgm_g.active = true
			  )`, groupArgN, wsArgN), []any{groupID}
}

func lockParentAndSheet(ctx context.Context, exec sqlexec.DBExecutor, templateID, phaseID, wsID, groupID string) ([]lockedPhase, error) {
	if templateID == "" || phaseID == "" {
		return nil, fmt.Errorf("job_phase approval: job_template_id and job_template_phase_id are required")
	}
	var gotParent string
	if err := exec.QueryRowContext(ctx, jobPhaseParentLockSQL, phaseID, templateID, wsID).Scan(&gotParent); err != nil {
		if err == sql.ErrNoRows {
			// Indistinguishable (deliberately, to avoid an enumeration oracle) from
			// a foreign/mismatched template/phase or a cross-tenant template.
			return nil, fmt.Errorf("job_phase approval: sheet parent not found for template/phase in this workspace")
		}
		return nil, fmt.Errorf("job_phase approval: lock parent template phase: %w", err)
	}

	narrow, narrowArgs := groupNarrowPredicate(groupID, 4, 3)
	rows, err := exec.QueryContext(ctx, jobPhaseSheetLockSQL(narrow), append([]any{templateID, phaseID, wsID}, narrowArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("job_phase approval: lock sheet set: %w", err)
	}
	defer rows.Close()
	var set []lockedPhase
	for rows.Next() {
		var lp lockedPhase
		if err := rows.Scan(&lp.id, &lp.jobID, &lp.approvalStatus); err != nil {
			return nil, fmt.Errorf("job_phase approval: scan locked phase: %w", err)
		}
		set = append(set, lp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_phase approval: iterate locked phases: %w", err)
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("job_phase approval: sheet has no phases (empty target)")
	}
	return set, nil
}

// sheetHardFrozen is the reusable hard_frozen read (D5 / plan §4.4): a sheet is
// hard-frozen when ANY of its jobs has an active authoritative job summary OR its
// enclosing academic-year price_schedule is closed. Ground-truthed against the
// year-final-compute CLI freeze predicate. The render-gate 409 itself is P3; P2
// ships this predicate as a reusable read and enforces it on every transition.
func sheetHardFrozen(ctx context.Context, exec sqlexec.DBExecutor, templateID, phaseID, wsID, groupID string) (bool, error) {
	narrow, narrowArgs := groupNarrowPredicate(groupID, 4, 3)
	q := `
		SELECT EXISTS (
			SELECT 1
			FROM ` + entityid.JobPhase + ` jp
			JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
			WHERE jp.template_phase_id = $2
			  AND j.job_template_id = $1
			  AND j.workspace_id = $3
			  AND jp.active = true` + narrow + `
			  AND (
			    EXISTS (
			      SELECT 1 FROM ` + entityid.JobOutcomeSummary + ` jos
			      WHERE jos.job_id = j.id AND jos.workspace_id = j.workspace_id
			        AND jos.active = true AND jos.is_authoritative = true
			    )
			    OR EXISTS (
			      SELECT 1 FROM ` + entityid.SubscriptionGroupMember + ` sgm
			      JOIN ` + entityid.SubscriptionGroup + ` sg ON sg.id = sgm.subscription_group_id
			      JOIN ` + entityid.PriceSchedule + ` ps ON ps.id = sg.price_schedule_id
			      WHERE j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
			        AND sgm.subscription_id = j.origin_id
			        AND sgm.workspace_id = j.workspace_id
			        AND sg.workspace_id = j.workspace_id
			        AND ps.workspace_id = j.workspace_id
			        AND ps.closed = true
			    )
			  )
		)`
	var frozen bool
	if err := exec.QueryRowContext(ctx, q, append([]any{templateID, phaseID, wsID}, narrowArgs...)...).Scan(&frozen); err != nil {
		return false, fmt.Errorf("job_phase approval: hard_frozen probe: %w", err)
	}
	return frozen, nil
}

// resolveStaffFacet resolves the ONE trusted active staff facet the submitter is
// acting as (codex "Exact partial-reachability result" §3; §2 HIGH re-proof). The
// facet is ALWAYS re-proven against the `staff` table INSIDE the transition
// transaction as exactly one active row matching the trusted workspace AND the
// authenticated user — the session's stamped PrincipalID/PrincipalType is NOT
// trusted on its own, so a staff row disabled (or re-homed to another user /
// workspace) after session creation is correctly rejected. For a STAFF persona
// the row is additionally pinned to id = PrincipalID; for an operator persona no
// staff-id is pinned and the (user, workspace) pair must resolve to exactly one
// active staff row. Missing/ambiguous/mismatched fails closed.
func resolveStaffFacet(ctx context.Context, exec sqlexec.DBExecutor, wsID string) (string, error) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.UserID == "" || wsID == "" {
		return "", fmt.Errorf("job_phase submit: no trusted identity to resolve a staff facet")
	}

	staffPrincipalID, isStaff := principalscope.StaffRowScope(ctx)
	if isStaff && staffPrincipalID == "" {
		// A malformed STAFF session (kind STAFF, empty principal id) fails closed.
		return "", fmt.Errorf("job_phase submit: staff session has no resolved principal id — fail closed")
	}

	// Re-prove exactly one active staff row bound to the authenticated user and
	// trusted workspace. $3 pins the STAFF persona's row to its principal id; an
	// operator persona passes NULL so the (user, workspace) pair alone must be
	// unambiguous.
	const q = `SELECT id FROM ` + entityid.Staff + `
		WHERE user_id = $1 AND workspace_id = $2 AND active = true
		  AND ($3::text IS NULL OR id = $3)`
	var pinned any
	if isStaff {
		pinned = staffPrincipalID
	} else {
		pinned = nil
	}
	rows, err := exec.QueryContext(ctx, q, id.UserID, wsID, pinned)
	if err != nil {
		return "", fmt.Errorf("job_phase submit: re-prove staff facet: %w", err)
	}
	defer rows.Close()
	var facet string
	n := 0
	for rows.Next() {
		n++
		if n > 1 {
			return "", fmt.Errorf("job_phase submit: ambiguous staff facet (multiple active staff rows for the acting user) — fail closed")
		}
		if err := rows.Scan(&facet); err != nil {
			return "", fmt.Errorf("job_phase submit: scan staff facet: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("job_phase submit: iterate staff facet: %w", err)
	}
	if n == 0 || facet == "" {
		return "", fmt.Errorf("job_phase submit: no active staff facet matching the acting user + trusted workspace (disabled/re-homed since login?) — fail closed")
	}
	return facet, nil
}

// assertAllTasksOwned enforces the D7 strict ownership contract over the FULL
// sheet: every active job_task under every phase member has assigned_to == facet,
// AND every phase member has at least one active task. Universal ownership is
// strictly stronger than existential reachability, so it also proves S\R=∅ (the
// codex "reachability is evidence only, at minimum S\R=∅" bar) — a substitute who
// merely recorded one outcome does NOT own all tasks and is correctly denied.
func assertAllTasksOwned(ctx context.Context, exec sqlexec.DBExecutor, templateID, phaseID, wsID, facet, groupID string) error {
	narrow, narrowArgs := groupNarrowPredicate(groupID, 5, 3)
	// (A) any active task not owned by the facet (unassigned / blank / other staff)?
	unownedSQL := `
		SELECT EXISTS (
			SELECT 1
			FROM ` + entityid.JobPhase + ` jp
			JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
			JOIN ` + entityid.JobTask + ` jt ON jt.job_phase_id = jp.id AND jt.active = true
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
			  AND jp.active = true` + narrow + `
			  AND (jt.assigned_to IS NULL OR jt.assigned_to = '' OR jt.assigned_to <> $4)
		)`
	var unowned bool
	if err := exec.QueryRowContext(ctx, unownedSQL, append([]any{templateID, phaseID, wsID, facet}, narrowArgs...)...).Scan(&unowned); err != nil {
		return fmt.Errorf("job_phase submit: ownership probe: %w", err)
	}
	if unowned {
		return fmt.Errorf("job_phase submit: not all active tasks are assigned to the acting staff (D7 ownership) — fail closed")
	}
	// (B) any phase member with zero active tasks (cannot prove ownership)?
	emptyNarrow, emptyNarrowArgs := groupNarrowPredicate(groupID, 4, 3)
	emptyMemberSQL := `
		SELECT EXISTS (
			SELECT 1
			FROM ` + entityid.JobPhase + ` jp
			JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
			  AND jp.active = true` + emptyNarrow + `
			  AND NOT EXISTS (SELECT 1 FROM ` + entityid.JobTask + ` jt WHERE jt.job_phase_id = jp.id AND jt.active = true)
		)`
	var emptyMember bool
	if err := exec.QueryRowContext(ctx, emptyMemberSQL, append([]any{templateID, phaseID, wsID}, emptyNarrowArgs...)...).Scan(&emptyMember); err != nil {
		return fmt.Errorf("job_phase submit: empty-member probe: %w", err)
	}
	if emptyMember {
		return fmt.Errorf("job_phase submit: a sheet phase has no active tasks — cannot prove ownership, fail closed")
	}
	return nil
}

// countBlankRequiredCells counts blank EFFECTIVE-REQUIRED MATRIX LEAVES across S
// (D6). The matrix grain is task × criterion, NOT task (codex §3 HIGH: the old
// task-level count let one populated criterion hide other blank required criteria,
// and collapsed several blanks to one). Codex P3 §A3/§B1 additionally: it must
// count ONLY effective-required leaves — join the active OutcomeCriteria and filter
// on COALESCE(ttc.required_override, oc.required) = true so optional leaves are NOT
// counted as blank and a per-template-task true/false required_override wins over
// the criterion default (template_task_criteria.required_override = 7,
// outcome_criteria.required = 30). This mirrors the outcome-matrix cells query's
// criterion join (outcome_matrix_query.go); each active job_task × active
// template_task_criteria pair is one leaf, blank when no active task_outcome matches
// BOTH the task and that criterion (t.criteria_version_id = ttc.outcome_criteria_id).
// Surfaced in P3's confirm dialog and stamped on the transition audit event.
func countBlankRequiredCells(ctx context.Context, exec sqlexec.DBExecutor, templateID, phaseID, wsID, groupID string) (int, error) {
	narrow, narrowArgs := groupNarrowPredicate(groupID, 4, 3)
	q := `
		SELECT COUNT(*)
		FROM ` + entityid.JobPhase + ` jp
		JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
		JOIN ` + entityid.JobTask + ` jt ON jt.job_phase_id = jp.id AND jt.active = true
		JOIN ` + entityid.TemplateTaskCriteria + ` ttc
		  ON ttc.job_template_task_id = jt.template_task_id AND ttc.active = true
		JOIN ` + entityid.OutcomeCriteria + ` oc
		  ON oc.id = ttc.outcome_criteria_id AND oc.active = true
		WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
		  AND jp.active = true` + narrow + `
		  AND COALESCE(ttc.required_override, oc.required) = true
		  AND NOT EXISTS (
		    SELECT 1 FROM ` + entityid.TaskOutcome + ` t
		    WHERE t.job_task_id = jt.id
		      AND t.criteria_version_id = ttc.outcome_criteria_id
		      AND t.active = true
		  )`
	var n int
	if err := exec.QueryRowContext(ctx, q, append([]any{templateID, phaseID, wsID}, narrowArgs...)...).Scan(&n); err != nil {
		return 0, fmt.Errorf("job_phase submit: blank-count probe: %w", err)
	}
	return n, nil
}

// bulkUpdateAndVerify runs the RETURNING-id bulk update and proves the sorted
// returned id set equals the sorted locked id set (a count compare is
// insufficient — codex). setSQL is the SET clause; args are its bound values
// starting at $2 ($1 is reserved for the id array).
func bulkUpdateAndVerify(ctx context.Context, exec sqlexec.DBExecutor, locked []lockedPhase, setSQL, wherePredicate string, args ...any) (int, error) {
	ids := make([]string, len(locked))
	for i, lp := range locked {
		ids[i] = lp.id
	}
	q := `UPDATE ` + entityid.JobPhase + ` SET ` + setSQL +
		` WHERE id = ANY($1)` + wherePredicate + ` RETURNING id`
	callArgs := append([]any{pq.Array(ids)}, args...)
	rows, err := exec.QueryContext(ctx, q, callArgs...)
	if err != nil {
		return 0, fmt.Errorf("job_phase approval: bulk update: %w", err)
	}
	defer rows.Close()
	var updated []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("job_phase approval: scan updated id: %w", err)
		}
		updated = append(updated, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("job_phase approval: iterate updated ids: %w", err)
	}
	sort.Strings(ids)
	sort.Strings(updated)
	if len(updated) != len(ids) {
		return 0, fmt.Errorf("job_phase approval: transition wrote %d of %d locked rows — source-state changed, rolling back", len(updated), len(ids))
	}
	for i := range ids {
		if ids[i] != updated[i] {
			return 0, fmt.Errorf("job_phase approval: returned id set differs from locked id set — rolling back")
		}
	}
	return len(updated), nil
}

// writeTransitionAudit appends ONE append-only sheet transition event in the same
// business transaction (codex "Audit snapshot and retention contract").
//
// AUDIT IS MANDATORY (codex §7 HIGH): a nil audit dependency FAILS the transition
// — a raw/custom transaction-capable repository can no longer transition without a
// durable event. The actor is derived DIRECTLY from the trusted authenticated
// identity at this boundary and stamped into the audit context, so the durable
// event records the authenticated user even when the shipped HTTP chain did not
// install the audit-context middleware (which otherwise leaves LogEntry inserting
// an empty/zero actor). A missing trusted actor also fails closed. Partition /
// insert errors from LogEntry propagate and stay transaction-fatal.
func (r *PostgresJobPhaseRepository) writeTransitionAudit(ctx context.Context, wsID, templateID, phaseID, groupID, permCode, useCase, oldState, newState, reason string, affected, blankCount int) error {
	if r.audit == nil {
		return fmt.Errorf("job_phase approval: audit dependency is absent — refusing to transition without a durable audit event (fail closed)")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.UserID == "" {
		return fmt.Errorf("job_phase approval: no trusted actor identity for the audit event (fail closed)")
	}
	// Derive/override the audit actor from trusted identity — never trust an empty
	// or mismatched middleware-supplied actor for this security-critical event.
	// Preserve any IP/UserAgent/RequestID the middleware did set.
	ac, _ := infraports.GetAuditContext(ctx)
	ac.ActorID = id.UserID
	ac.ActorType = "user"
	ctx = infraports.WithAuditContext(ctx, ac)

	changes := []infraports.AuditFieldChange{
		{FieldName: "approval_status", FieldType: 1, OldValue: oldState, NewValue: newState},
		{FieldName: "job_template_id", FieldType: 1, OldValue: "", NewValue: templateID},
		{FieldName: "affected_count", FieldType: 1, OldValue: "", NewValue: strconv.Itoa(affected)},
	}
	if blankCount >= 0 {
		changes = append(changes, infraports.AuditFieldChange{
			FieldName: "blank_required_cell_count", FieldType: 1, OldValue: "", NewValue: strconv.Itoa(blankCount),
		})
	}
	// Record the narrowing. Without it a group-scoped transition is
	// indistinguishable in the trail from a template-wide one that happened to
	// affect the same count — which is exactly the question an auditor asks
	// ("who submitted whose grades?").
	if groupID != "" {
		changes = append(changes, infraports.AuditFieldChange{
			FieldName: "subscription_group_id", FieldType: 1, OldValue: "", NewValue: groupID,
		})
	}
	return r.audit.LogEntry(ctx, &infraports.AuditLogRequest{
		WorkspaceID:    wsID,
		EntityType:     entityid.JobPhase,
		EntityID:       phaseID, // the sheet key = parent job_template_phase id
		Domain:         "espyna",
		Action:         2, // AUDIT_ACTION_UPDATE
		PermissionCode: permCode,
		UseCase:        useCase,
		Reason:         reason,
		MethodName:     useCase,
		FieldChanges:   changes,
	})
}

// deriveOldState returns the sole approval status shared by every locked row, or
// "MIXED" when the captured statuses actually differ (codex §7 HIGH: Return must
// record its ACTUAL old state, not an assumed "MIXED" even for a uniformly
// FOR_REVIEW / VERIFIED / PUBLISHED sheet). Empty set → empty string.
func deriveOldState(locked []lockedPhase) string {
	if len(locked) == 0 {
		return ""
	}
	first := locked[0].approvalStatus
	for _, lp := range locked[1:] {
		if lp.approvalStatus != first {
			return "MIXED"
		}
	}
	return first
}

// requireUniformSource asserts every locked row is in the expected source state.
func requireUniformSource(locked []lockedPhase, want string) error {
	for _, lp := range locked {
		if lp.approvalStatus != want {
			return fmt.Errorf("job_phase approval: sheet is not uniformly %s (found %s) — mixed or wrong source state", want, lp.approvalStatus)
		}
	}
	return nil
}

func nowMillis() int64 { return time.Now().UnixMilli() }

// submitFreshnessBarrier is the FIX-3 seam Submit invokes between the D6 blank
// count and the status flip: it derives the locked sheet's phase ids + distinct job
// ids and invokes the injected recompute port on the AMBIENT transaction context.
// A nil port fails closed (a raw/custom repository cannot submit without the
// barrier); a recompute error fails the submit (the caller returns it, so the
// surrounding transaction rolls back). Extracted so the fail-closed/failure
// contract is unit-testable without a database.
// verb names the transition in error messages ("submit", "verify") so a barrier
// failure says which transition rolled back. Both call sites share one contract:
// recompute on the ambient tx, post-lock/pre-flip, fail closed.
func submitFreshnessBarrier(ctx context.Context, verb string, recompute func(ctx context.Context, phaseIDs, jobIDs []string) error, locked []lockedPhase) error {
	if recompute == nil {
		return fmt.Errorf("job_phase %s: summary-recompute barrier is not wired — refusing to advance a possibly-stale sheet (fail closed)", verb)
	}
	phaseIDs := make([]string, 0, len(locked))
	jobIDs := make([]string, 0, len(locked))
	seenJob := make(map[string]struct{}, len(locked))
	for _, lp := range locked {
		phaseIDs = append(phaseIDs, lp.id)
		if lp.jobID == "" {
			continue
		}
		if _, ok := seenJob[lp.jobID]; ok {
			continue
		}
		seenJob[lp.jobID] = struct{}{}
		jobIDs = append(jobIDs, lp.jobID)
	}
	if rerr := recompute(ctx, phaseIDs, jobIDs); rerr != nil {
		return fmt.Errorf("job_phase %s: freshness-barrier recompute failed — rolling back: %w", verb, rerr)
	}
	return nil
}

// ---- SubmitJobPhaseApproval: IN_PROGRESS -> FOR_REVIEW ----

func (r *PostgresJobPhaseRepository) SubmitJobPhaseApproval(ctx context.Context, req *pb.SubmitJobPhaseApprovalRequest) (*pb.SubmitJobPhaseApprovalResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("job_phase submit: request is required")
	}
	exec, err := r.txExecutor(ctx)
	if err != nil {
		return nil, err
	}
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil, fmt.Errorf("job_phase submit: no workspace in trusted context")
	}
	templateID, phaseID := req.GetJobTemplateId(), req.GetJobTemplatePhaseId()
	// Optional delivery-group narrowing. Empty ⇒ the whole template, unchanged.
	groupID := req.GetSubscriptionGroupId()

	locked, err := lockParentAndSheet(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}

	// Actor authorization (D7). Admin override (proven job_phase:publish authority)
	// resolved by the use case and threaded via context; default fail-closed to
	// the strict all-task ownership check.
	decision, _ := approvalctx.SubmitDecisionFromContext(ctx)
	if !decision.AdminOverride {
		facet, ferr := resolveStaffFacet(ctx, exec, wsID)
		if ferr != nil {
			return nil, ferr
		}
		if aerr := assertAllTasksOwned(ctx, exec, templateID, phaseID, wsID, facet, groupID); aerr != nil {
			return nil, aerr
		}
	}

	if err := requireUniformSource(locked, apInProgress); err != nil {
		return nil, err
	}
	frozen, err := sheetHardFrozen(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if frozen {
		return nil, fmt.Errorf("job_phase submit: sheet is hard-frozen (closed schedule or authoritative final) — cannot submit")
	}

	blankCount, err := countBlankRequiredCells(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}

	// FIX-3 submit-time freshness barrier (codex-rereview.md "Exact lock and state
	// contract" step 4 + FIX-FIRST 3 "cannot be deferred"): finalize the phase then
	// job outcome summaries for the locked sheet INSIDE this transaction, under the
	// parent FOR UPDATE mutex, BEFORE flipping IN_PROGRESS → FOR_REVIEW. Because the
	// recompute runs on the ambient tx (r.recompute delegates to grade_compute on
	// GetExecutor(ctx) = this *sql.Tx), a recompute failure fails the whole transition
	// (tx rollback). A gradable-but-blank or non-gradable phase is an expected skip
	// (D6 permits partial/blank submission), so the barrier does NOT reject a blank
	// sheet — only a genuine compute/read failure.
	//
	// The barrier is coherent with the FIX-2 cell-write guard: a cell save takes the
	// parent FOR SHARE (guardCellWrite step 2, job_phase_approval.go) which is
	// incompatible with this submit's parent FOR UPDATE, so any in-flight edit has
	// already committed (its values are what we finalize here) or blocks until this
	// transition commits and then fails guardCellWrite's IN_PROGRESS recheck (step 4).
	// A "delayed" recompute triggered by a post-submit cell save therefore never runs
	// (the leaf write is rejected first), so it cannot overwrite these finalized
	// summaries with stale input once approval has advanced.
	if err := submitFreshnessBarrier(ctx, "submit", r.recompute, locked); err != nil {
		return nil, err
	}

	actor := identity.Must(ctx).UserID
	ts := nowMillis()
	// Submit starts a new cycle: overwrite submitted pair; clear verified,
	// published, returned pairs and reason.
	setSQL := `approval_status = '` + apForReview + `',
		submitted_by = $2, submitted_at = $3,
		verified_by = NULL, verified_at = NULL,
		published_by = NULL, published_at = NULL,
		returned_by = NULL, returned_at = NULL, return_reason = NULL,
		date_modified = now()`
	affected, err := bulkUpdateAndVerify(ctx, exec, locked, setSQL, ` AND approval_status = '`+apInProgress+`'`, actor, ts)
	if err != nil {
		return nil, err
	}
	if err := r.writeTransitionAudit(ctx, wsID, templateID, phaseID, groupID, "job_phase:submit", "SubmitJobPhaseApproval", apInProgress, apForReview, "", affected, blankCount); err != nil {
		return nil, err
	}
	return &pb.SubmitJobPhaseApprovalResponse{
		Status:        pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_FOR_REVIEW,
		AffectedCount: int32(affected),
		Success:       true,
	}, nil
}

// ---- VerifyJobPhaseApproval: FOR_REVIEW -> VERIFIED ----

func (r *PostgresJobPhaseRepository) VerifyJobPhaseApproval(ctx context.Context, req *pb.VerifyJobPhaseApprovalRequest) (*pb.VerifyJobPhaseApprovalResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("job_phase verify: request is required")
	}
	exec, err := r.txExecutor(ctx)
	if err != nil {
		return nil, err
	}
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil, fmt.Errorf("job_phase verify: no workspace in trusted context")
	}
	templateID, phaseID := req.GetJobTemplateId(), req.GetJobTemplatePhaseId()
	// Optional delivery-group narrowing. Empty ⇒ the whole template, unchanged.
	groupID := req.GetSubscriptionGroupId()

	locked, err := lockParentAndSheet(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if err := requireUniformSource(locked, apForReview); err != nil {
		return nil, err
	}
	frozen, err := sheetHardFrozen(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if frozen {
		return nil, fmt.Errorf("job_phase verify: sheet is hard-frozen — cannot verify")
	}

	// Verify-time freshness barrier: re-finalize the phase then job outcome
	// summaries for the locked sheet INSIDE this transaction, under the parent
	// FOR UPDATE mutex, BEFORE flipping FOR_REVIEW → VERIFIED. Identical contract
	// to the submit barrier (ambient tx, fail closed, rollback on error).
	//
	// Between FOR_REVIEW and VERIFIED the sheet is locked, so in the normal UI flow
	// this recomputes the same inputs submit already finalized and is a no-op. Its
	// value is as an ASSERTION: if anything mutated the sheet out-of-band (a direct
	// DB write, a grade-loader/CLI run, an import), verify fails closed and rolls
	// back rather than blessing a stale summary as verified. A verified summary is
	// therefore provably derived from the data present at verification time.
	if err := submitFreshnessBarrier(ctx, "verify", r.recompute, locked); err != nil {
		return nil, err
	}

	actor := identity.Must(ctx).UserID
	ts := nowMillis()
	// Verify preserves the submitted pair; sets the verified pair.
	setSQL := `approval_status = '` + apVerified + `', verified_by = $2, verified_at = $3, date_modified = now()`
	affected, err := bulkUpdateAndVerify(ctx, exec, locked, setSQL, ` AND approval_status = '`+apForReview+`'`, actor, ts)
	if err != nil {
		return nil, err
	}
	if err := r.writeTransitionAudit(ctx, wsID, templateID, phaseID, groupID, "job_phase:verify", "VerifyJobPhaseApproval", apForReview, apVerified, "", affected, -1); err != nil {
		return nil, err
	}
	return &pb.VerifyJobPhaseApprovalResponse{
		Status:        pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED,
		AffectedCount: int32(affected),
		Success:       true,
	}, nil
}

// ---- PublishJobPhaseApproval: VERIFIED -> PUBLISHED ----

func (r *PostgresJobPhaseRepository) PublishJobPhaseApproval(ctx context.Context, req *pb.PublishJobPhaseApprovalRequest) (*pb.PublishJobPhaseApprovalResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("job_phase publish: request is required")
	}
	exec, err := r.txExecutor(ctx)
	if err != nil {
		return nil, err
	}
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil, fmt.Errorf("job_phase publish: no workspace in trusted context")
	}
	templateID, phaseID := req.GetJobTemplateId(), req.GetJobTemplatePhaseId()
	// Optional delivery-group narrowing. Empty ⇒ the whole template, unchanged.
	groupID := req.GetSubscriptionGroupId()

	locked, err := lockParentAndSheet(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if err := requireUniformSource(locked, apVerified); err != nil {
		return nil, err
	}
	frozen, err := sheetHardFrozen(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if frozen {
		return nil, fmt.Errorf("job_phase publish: sheet is hard-frozen — cannot publish")
	}

	actor := identity.Must(ctx).UserID
	ts := nowMillis()
	// Publish preserves submitted/verified pairs; sets the published pair.
	setSQL := `approval_status = '` + apPublished + `', published_by = $2, published_at = $3, date_modified = now()`
	affected, err := bulkUpdateAndVerify(ctx, exec, locked, setSQL, ` AND approval_status = '`+apVerified+`'`, actor, ts)
	if err != nil {
		return nil, err
	}
	if err := r.writeTransitionAudit(ctx, wsID, templateID, phaseID, groupID, "job_phase:publish", "PublishJobPhaseApproval", apVerified, apPublished, "", affected, -1); err != nil {
		return nil, err
	}
	return &pb.PublishJobPhaseApprovalResponse{
		Status:        pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED,
		AffectedCount: int32(affected),
		Success:       true,
	}, nil
}

// ---- ReturnJobPhaseApproval: mixed normalizer -> IN_PROGRESS ----

func (r *PostgresJobPhaseRepository) ReturnJobPhaseApproval(ctx context.Context, req *pb.ReturnJobPhaseApprovalRequest) (*pb.ReturnJobPhaseApprovalResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("job_phase return: request is required")
	}
	exec, err := r.txExecutor(ctx)
	if err != nil {
		return nil, err
	}
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil, fmt.Errorf("job_phase return: no workspace in trusted context")
	}
	templateID, phaseID := req.GetJobTemplateId(), req.GetJobTemplatePhaseId()
	// Optional delivery-group narrowing. Empty ⇒ the whole template, unchanged.
	groupID := req.GetSubscriptionGroupId()

	locked, err := lockParentAndSheet(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}

	// Return is the mixed-state normalizer: require >= 1 non-IN_PROGRESS row.
	anyAdvanced := false
	anyPublished := false
	for _, lp := range locked {
		if lp.approvalStatus != apInProgress {
			anyAdvanced = true
		}
		if lp.approvalStatus == apPublished {
			anyPublished = true
		}
	}
	if !anyAdvanced {
		return nil, fmt.Errorf("job_phase return: sheet is already fully IN_PROGRESS — nothing to return")
	}

	// A row that is or WAS published (published_by stamped) also requires a reason.
	// Probe published_by so a return of a since-mixed sheet whose published rows
	// were normalized still requires the reason.
	if !anyPublished {
		const wasPublishedSQL = `
			SELECT EXISTS (
				SELECT 1 FROM ` + entityid.JobPhase + ` jp
				JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
				WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
				  AND jp.active = true AND jp.published_by IS NOT NULL
			)`
		if err := exec.QueryRowContext(ctx, wasPublishedSQL, templateID, phaseID, wsID).Scan(&anyPublished); err != nil {
			return nil, fmt.Errorf("job_phase return: was-published probe: %w", err)
		}
	}

	reason, err := normalizeReturnReason(req.GetReason(), anyPublished)
	if err != nil {
		return nil, err
	}

	// Hard-frozen blocks return entirely (a closed schedule / authoritative final
	// needs a separate correction workflow).
	frozen, err := sheetHardFrozen(ctx, exec, templateID, phaseID, wsID, groupID)
	if err != nil {
		return nil, err
	}
	if frozen {
		return nil, fmt.Errorf("job_phase return: sheet is hard-frozen — return is blocked, use a correction workflow")
	}

	actor := identity.Must(ctx).UserID
	ts := nowMillis()
	// Return preserves prior ladder stamps (submitted/verified/published pairs) so
	// support can see what was returned; sets the returned pair + reason; status
	// becomes IN_PROGRESS. reason is stored NULL when blank.
	var reasonArg any
	if reason == "" {
		reasonArg = nil
	} else {
		reasonArg = reason
	}
	setSQL := `approval_status = '` + apInProgress + `', returned_by = $2, returned_at = $3, return_reason = $4, date_modified = now()`
	affected, err := bulkUpdateAndVerify(ctx, exec, locked, setSQL, ` AND active = true`, actor, ts, reasonArg)
	if err != nil {
		return nil, err
	}
	// Record the ACTUAL old state (the sole captured status, or MIXED only when
	// the locked statuses genuinely differ) — never an assumed "MIXED".
	oldState := deriveOldState(locked)
	if err := r.writeTransitionAudit(ctx, wsID, templateID, phaseID, groupID, "job_phase:return", "ReturnJobPhaseApproval", oldState, apInProgress, reason, affected, -1); err != nil {
		return nil, err
	}
	return &pb.ReturnJobPhaseApprovalResponse{
		Status:        pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS,
		AffectedCount: int32(affected),
		Success:       true,
	}, nil
}
