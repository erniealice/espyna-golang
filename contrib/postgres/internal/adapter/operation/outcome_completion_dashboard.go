//go:build postgresql

// outcome_completion_dashboard.go — the live, cell-grain outcome-completion
// aggregate feeding the persona-aware home dashboard
// (docs/plan/20260801-persona-home-dashboard plan.md §4.3, Phase 2; Q3/Q5/Q7
// all LOCKED 2026-08-01 INCLUDING the §4.3 owner-acked amendment banner).
//
// NEW FILE by contract: `principalscope.go` is mirror-only (plan.md §9) — the
// staff scoping below re-derives the reachable-set tiers from
// principalscope.reachableJobUnion and then applies the A1 most-specific-wins
// responsibility predicate ON TOP, which the shared union deliberately does
// not know about. Keep the tier shapes in step with principalscope.go.
package operation

import (
	"context"
	"database/sql"
	"fmt"

	ocdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/outcome_completion"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS named-type contract: alias to the
// service-layer row type so Go's exact-named-return-type matching makes the
// repository assertion below succeed (the Wave B P1.C.1 nil-repo trap guard).
type CategoryPeriodCount = ocdash.CategoryPeriodCount

// Compile-time guard: the postgres job adapter satisfies the service-layer
// outcome-completion dashboard repository interface. The composition root
// (internal/composition/core/initializers/service/dashboard.go) type-asserts
// operationRepos.Job into this interface at runtime; without this line a
// signature drift would silently leave the repo nil (the P1.C.1 bug).
var _ ocdash.OutcomeCompletionSummaryRepository = (*PostgresJobRepository)(nil)

// Approval-ladder tokens as persisted by job_phase.approval_status (the enum
// NAME, TEXT NOT NULL — job_phase.proto). Local consts; the rank CASE in
// job_template_summary_query.go inlines the same literals.
const (
	ocApprovalInProgress = "PHASE_APPROVAL_STATUS_IN_PROGRESS"
	ocApprovalForReview  = "PHASE_APPROVAL_STATUS_FOR_REVIEW"
	ocApprovalVerified   = "PHASE_APPROVAL_STATUS_VERIFIED"
	ocApprovalPublished  = "PHASE_APPROVAL_STATUS_PUBLISHED"
)

// currentPeriodGroupStatus is the free-text subscription_group.status token
// that marks the CURRENT enrollment window (the membership-anchored current
// window: subscription_group status axis, 20260726). The aggregate's job set
// is confined to jobs whose origin subscription holds an active membership in
// an active, current group — the same window the storybook-findings F3/F6
// verification measured.
const currentPeriodGroupStatus = "current"

// outcomeCompletionKinds — the principal kinds this aggregate defines.
// Mirrors principalscope.PrincipalTypeStaff (7) plus the operator kinds; any
// OTHER kind (including the unresolved 0 sentinel and client/supplier/delegate
// kinds) is fail-closed to zero rows here even though the use case already
// denies it — defense in depth.
const (
	ocKindOperatorOwner int32 = 1
	ocKindOperatorStaff int32 = 2
)

// outcomeCompletionCellScope returns the responsibility predicate spliced into
// the cells CTE (aliases in scope: j = job, jp = job_phase, tk = job_task),
// with the positional bind args to append in order. startParam is the 1-based
// index of the NEXT placeholder.
//
//   - operator kinds 1/2      → ("", nil): workspace grain, query unchanged.
//   - staff kind 7, real id   → (A1 predicate, []any{staffID, sessionWorkspaceID}).
//   - staff kind 7, empty id  → (" AND 1=0", nil): fail-closed, zero rows.
//   - no identity / any other → (" AND 1=0", nil): fail-closed (kind 0 included).
//
// **A1 — most-specific-wins responsibility (owner-acked amendment, BINDING).**
// Each cell's responsible staff is resolved by the MOST SPECIFIC live edge;
// less specific tiers are outranked and cannot claim the cell:
//
//  1. task grain — job_task.assigned_to names a staff: only that staff owns
//     the task's cells.
//  2. phase grain — a live phase-scoped class edge (sgpps with
//     job_template_phase_id, f14) claims exactly its phase's cells
//     (job_phase.template_phase_id match). A rotation teacher's denominator
//     therefore EXCLUDES sibling-semester strand cells (the storybook-findings
//     F2 worked example: Design Sem-2 must not count Sem-1 strand cells), and
//     symmetrically her phase's cells are blocked from every job-grain
//     claimant (the NOT EXISTS sibling guard).
//  3. job grain — the mirrored reachable-job union tiers (task_outcome
//     recorded_by/reviewed_by; deliverable-matched subscription_seat;
//     phase-BLANK class edge = f14 NULL, "all phases") claim a phase's cells
//     only when no phase-scoped edge (any staff) covers that phase and the
//     task is unassigned.
//
// Class-edge staff resolution is v2-native, mirroring principalscope:
// COALESCE(pps.staff_id, e.staff_id) with the eligibility-liveness guard
// (a LINKED-but-REVOKED product_plan_staff row must not fall through to the
// legacy f10 column). The session workspace bound rides on every edge tier
// (belt-and-suspenders — the scope cannot be widened by a request param).
func outcomeCompletionCellScope(ctx context.Context, startParam int) (clause string, args []any) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil {
		return " AND 1=0", nil
	}
	switch id.PrincipalType {
	case ocKindOperatorOwner, ocKindOperatorStaff:
		return "", nil
	case principalscopeStaffKind:
		// handled below
	default:
		return " AND 1=0", nil
	}
	if id.PrincipalID == "" {
		return " AND 1=0", nil
	}

	s := fmt.Sprintf("$%d", startParam)
	w := fmt.Sprintf("$%d", startParam+1)

	// Phase-scoped class-edge EXISTS, parameterized by alias suffix and an
	// optional staff match (the sibling guard omits it). An equality on
	// e.job_template_phase_id is NULL-safe: a NULL f14 (job-grain edge) or a
	// NULL jp.template_phase_id never matches.
	phaseEdge := func(sfx, staffMatch string) string {
		return "SELECT 1 FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" + sfx +
			" JOIN " + entityid.SubscriptionGroupMember + " m" + sfx +
			" ON m" + sfx + ".subscription_group_id = e" + sfx + ".subscription_group_id AND m" + sfx + ".active" +
			" JOIN " + entityid.ProductPlan + " pp" + sfx + " ON pp" + sfx + ".id = e" + sfx + ".product_plan_id" +
			" LEFT JOIN " + entityid.ProductPlanStaff + " pps" + sfx + " ON pps" + sfx + ".id = e" + sfx + ".product_plan_staff_id" +
			" WHERE m" + sfx + ".subscription_id = j.origin_id" +
			" AND pp" + sfx + ".product_id = j.output_product_id" +
			" AND e" + sfx + ".job_template_phase_id = jp.template_phase_id" +
			staffMatch +
			" AND (e" + sfx + ".product_plan_staff_id IS NULL OR pps" + sfx + ".active)" +
			" AND e" + sfx + ".active AND e" + sfx + ".workspace_id = " + w
	}

	clause = " AND (" +
		// Tier 1 — task grain: an explicit assignee owns the cell outright.
		"tk.assigned_to = " + s +
		" OR (NULLIF(tk.assigned_to, '') IS NULL AND (" +
		// Tier 2 — the acting staff's own phase-scoped edge covers THIS phase.
		"EXISTS (" + phaseEdge("", " AND COALESCE(pps.staff_id, e.staff_id) = "+s) + ")" +
		" OR (" +
		// Sibling guard: no phase-scoped live edge (ANY staff) covers this
		// phase — otherwise the phase's cells belong to those edges only.
		"NOT EXISTS (" + phaseEdge("2", "") + ")" +
		" AND (" +
		// Tier 3a — delivery axis (mirrors reachableJobUnion tier 2): the
		// staff recorded/reviewed an outcome anywhere on this job.
		"EXISTS (SELECT 1 FROM " + entityid.JobPhase + " jpo" +
		" JOIN " + entityid.JobTask + " tko ON tko.job_phase_id = jpo.id" +
		" JOIN " + entityid.TaskOutcome + " txo ON txo.job_task_id = tko.id" +
		" WHERE jpo.job_id = j.id AND (txo.recorded_by = " + s + " OR txo.reviewed_by = " + s + "))" +
		// Tier 3b — deliverable-matched subscription seat (mirrors tier 3).
		" OR EXISTS (SELECT 1 FROM " + entityid.SubscriptionSeat + " ss" +
		" JOIN " + entityid.JobTemplate + " tps ON tps.id = j.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tps.output_product_id" +
		" WHERE ss.subscription_id = j.origin_id AND ss.staff_id = " + s +
		" AND ss.status = 'active' AND ss.active = true AND ss.workspace_id = " + w + ")" +
		// Tier 3c — phase-BLANK class edge (f14 NULL = all phases; mirrors
		// reachableJobUnion tier 4 with the A1 phase restriction inapplicable).
		" OR EXISTS (SELECT 1 FROM " + entityid.SubscriptionGroupProductPlanStaff + " e3" +
		" JOIN " + entityid.SubscriptionGroupMember + " m3 ON m3.subscription_group_id = e3.subscription_group_id AND m3.active" +
		" JOIN " + entityid.ProductPlan + " pp3 ON pp3.id = e3.product_plan_id" +
		" LEFT JOIN " + entityid.ProductPlanStaff + " pps3 ON pps3.id = e3.product_plan_staff_id" +
		" WHERE m3.subscription_id = j.origin_id" +
		" AND pp3.product_id = j.output_product_id" +
		" AND e3.job_template_phase_id IS NULL" +
		" AND COALESCE(pps3.staff_id, e3.staff_id) = " + s +
		" AND (e3.product_plan_staff_id IS NULL OR pps3.active)" +
		" AND e3.active AND e3.workspace_id = " + w + ")" +
		")))" +
		"))"
	return clause, []any{id.PrincipalID, id.WorkspaceID}
}

// principalscopeStaffKind mirrors principalscope.PrincipalTypeStaff (7) —
// re-declared so this file needs no import of the sibling package's constant
// (the package itself is mirror-only by plan contract; keep the two in step).
const principalscopeStaffKind int32 = 7

// buildOutcomeCompletionSummarySQL is the pure SQL builder (no ctx, no DB —
// the permission_query_test.go testing idiom). ONE statement, fixed grouping
// and a FIXED ORDER BY (no interpolation surface):
//
//	$1         = workspaceID (bound on every workspace-carrying table)
//	scope args = scopeFn(2) result, spliced into the cells CTE
//
// Statement shape (two MATERIALIZED CTEs + derived aggregates — the
// courses-perf lesson: never flat-join 10 tables):
//
//	cells    MATERIALIZED — ONE row per expected cell (active job_task ×
//	         active template_task_criteria) over the CURRENT-window job set
//	         (active, workspace-bound, subscription-originated jobs whose
//	         origin subscription holds an active membership in an active
//	         subscription_group with status = 'current'), carrying the
//	         phase's identity/order/approval columns, the template's
//	         job_category_id, and recorded = EXISTS(active task_outcome for
//	         the cell's criteria version). Synthesized phase rows are
//	         excluded EXPLICITLY (NOT jp.is_synthesized — F5 ride-along),
//	         as are inactive rows at every join. The optional staff
//	         responsibility predicate (A1) splices in here.
//	cellagg  MATERIALIZED — (category, phase_order) cell counts: the A2
//	         period dimension at its finest response grain.
//	phases   — cells collapsed to phase grain (approval columns ride along;
//	         has_recorded = any recorded cell in the caller's scope).
//	phaseagg — (category, phase_order) DISJOINT approval-ladder counts:
//	         not_started / in_progress / for_review / verified / published /
//	         returned. returned = IN_PROGRESS with returned_at set (a return
//	         re-enters IN_PROGRESS); not_started = IN_PROGRESS with no
//	         submitted_at, no returned_at and no recorded cell (IN_PROGRESS
//	         is the spawn default); in_progress = the remaining IN_PROGRESS.
//	catrows  — every ACTIVE workspace category (so zero-cell categories still
//	         emit a row and the tab strip shares this single read) plus an
//	         uncategorized residue row when cells carry a NULL category.
//
// The final SELECT LEFT-joins the aggregates onto catrows with NULL-safe
// matches and a fixed ORDER BY (category sort_order asc, id asc, phase_order
// asc NULLS LAST).
func buildOutcomeCompletionSummarySQL(
	workspaceID string,
	scopeFn func(startParam int) (clause string, args []any),
) (stmt string, args []any) {
	args = []any{workspaceID}
	scopeClause := ""
	if scopeFn != nil {
		var scopeArgs []any
		scopeClause, scopeArgs = scopeFn(2)
		args = append(args, scopeArgs...)
	}

	stmt = `WITH cells AS MATERIALIZED (
    SELECT
        jp.id              AS phase_id,
        jp.phase_order     AS phase_order,
        jp.approval_status AS approval_status,
        jp.submitted_at    AS submitted_at,
        jp.returned_at     AS returned_at,
        jt.job_category_id AS category_id,
        EXISTS (
            SELECT 1 FROM ` + entityid.TaskOutcome + ` tox
            WHERE tox.job_task_id = tk.id
              AND tox.criteria_version_id = ttc.outcome_criteria_id
              AND tox.active
        ) AS recorded
    FROM ` + entityid.Job + ` j
    JOIN ` + entityid.JobTemplate + ` jt
           ON jt.id = j.job_template_id AND jt.workspace_id = $1 AND jt.active
    JOIN ` + entityid.JobPhase + ` jp
           ON jp.job_id = j.id AND jp.active AND NOT jp.is_synthesized
    JOIN ` + entityid.JobTask + ` tk
           ON tk.job_phase_id = jp.id AND tk.active
    JOIN ` + entityid.TemplateTaskCriteria + ` ttc
           ON ttc.job_template_task_id = tk.template_task_id AND ttc.active
    WHERE j.active
      AND j.workspace_id = $1
      AND j.origin_type = '` + originTypeSubscriptionToken + `'
      AND EXISTS (
          SELECT 1 FROM ` + entityid.SubscriptionGroupMember + ` cm
          JOIN ` + entityid.SubscriptionGroup + ` cg
                 ON cg.id = cm.subscription_group_id
                AND cg.workspace_id = $1 AND cg.active
                AND cg.status = '` + currentPeriodGroupStatus + `'
          WHERE cm.subscription_id = j.origin_id
            AND cm.client_id = j.client_id
            AND cm.workspace_id = $1 AND cm.active
      )` + scopeClause + `
),
cellagg AS MATERIALIZED (
    SELECT category_id, phase_order,
           COUNT(*)                         AS expected_cells,
           COUNT(*) FILTER (WHERE recorded) AS recorded_cells
    FROM cells
    GROUP BY category_id, phase_order
),
phases AS (
    SELECT phase_id, category_id, phase_order, approval_status, submitted_at, returned_at,
           BOOL_OR(recorded) AS has_recorded
    FROM cells
    GROUP BY phase_id, category_id, phase_order, approval_status, submitted_at, returned_at
),
phaseagg AS (
    SELECT category_id, phase_order,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalInProgress + `'
                              AND returned_at IS NULL AND submitted_at IS NULL
                              AND NOT has_recorded)                             AS not_started,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalInProgress + `'
                              AND returned_at IS NULL
                              AND (submitted_at IS NOT NULL OR has_recorded))   AS in_progress,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalForReview + `')  AS for_review,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalVerified + `')   AS verified,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalPublished + `')  AS published,
           COUNT(*) FILTER (WHERE approval_status = '` + ocApprovalInProgress + `'
                              AND returned_at IS NOT NULL)                      AS returned_rows
    FROM phases
    GROUP BY category_id, phase_order
),
catrows AS (
    SELECT cat.id AS category_id, cat.name AS category_name,
           COALESCE(cat.sort_order, 2147483647) AS category_sort
    FROM ` + entityid.JobCategory + ` cat
    WHERE cat.active AND cat.workspace_id = $1
    UNION ALL
    SELECT DISTINCT ca2.category_id, ''::text, 2147483647
    FROM cellagg ca2
    WHERE ca2.category_id IS NULL
)
SELECT
    cr.category_id,
    cr.category_name,
    ca.phase_order,
    COALESCE(ca.expected_cells, 0)  AS expected_cells,
    COALESCE(ca.recorded_cells, 0)  AS recorded_cells,
    COALESCE(pa.not_started, 0)     AS not_started,
    COALESCE(pa.in_progress, 0)     AS in_progress,
    COALESCE(pa.for_review, 0)      AS for_review,
    COALESCE(pa.verified, 0)        AS verified,
    COALESCE(pa.published, 0)       AS published,
    COALESCE(pa.returned_rows, 0)   AS returned_rows
FROM catrows cr
LEFT JOIN cellagg ca
       ON ca.category_id IS NOT DISTINCT FROM cr.category_id
LEFT JOIN phaseagg pa
       ON pa.category_id IS NOT DISTINCT FROM cr.category_id
      AND pa.phase_order = ca.phase_order
ORDER BY cr.category_sort ASC, cr.category_id ASC NULLS LAST, ca.phase_order ASC NULLS LAST`
	return stmt, args
}

// CompletionSummary runs the outcome-completion aggregate for one workspace.
// Row scoping comes from ctx identity (never a request param): operator kinds
// 1/2 read at workspace grain; staff kind 7 gets the A1 most-specific-wins
// responsibility predicate; anything else is fail-closed to zero rows. Errors
// PROPAGATE (T-9 — the view renders a designed error state, not silent zeros).
func (r *PostgresJobRepository) CompletionSummary(
	ctx context.Context,
	workspaceID string,
) ([]CategoryPeriodCount, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	stmt, args := buildOutcomeCompletionSummarySQL(workspaceID, func(start int) (string, []any) {
		return outcomeCompletionCellScope(ctx, start)
	})

	rows, err := r.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("outcome_completion: summary query: %w", err)
	}
	defer rows.Close()

	out := make([]CategoryPeriodCount, 0, 8)
	for rows.Next() {
		var (
			categoryID   sql.NullString
			categoryName sql.NullString
			phaseOrder   sql.NullInt32
			row          CategoryPeriodCount
		)
		if scanErr := rows.Scan(
			&categoryID, &categoryName, &phaseOrder,
			&row.ExpectedCells, &row.RecordedCells,
			&row.NotStarted, &row.InProgress, &row.ForReview,
			&row.Verified, &row.Published, &row.Returned,
		); scanErr != nil {
			return nil, fmt.Errorf("outcome_completion: scan summary row: %w", scanErr)
		}
		if categoryID.Valid {
			row.CategoryID = categoryID.String
		}
		if categoryName.Valid {
			row.CategoryName = categoryName.String
		}
		if phaseOrder.Valid {
			row.HasSlice = true
			row.PhaseOrder = phaseOrder.Int32
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outcome_completion: summary rows: %w", err)
	}
	return out, nil
}
