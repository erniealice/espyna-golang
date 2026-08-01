//go:build postgresql

package operation

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// ocCtx builds a ctx carrying a session RequestIdentity of the given kind.
func ocCtx(kind int32, principalID, workspaceID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID:        "user-1",
		WorkspaceID:   workspaceID,
		PrincipalType: kind,
		PrincipalID:   principalID,
	})
}

// ocStaffSQL builds the full statement under a kind-7 staff scope.
func ocStaffSQL(t *testing.T) (string, []any) {
	t.Helper()
	ctx := ocCtx(7, "staff-rot-1", "ws-session")
	stmt, args := buildOutcomeCompletionSummarySQL("ws-1", func(start int) (string, []any) {
		return outcomeCompletionCellScope(ctx, start)
	})
	return stmt, args
}

// TestOutcomeCompletionSQL_TableNamesFromEntityID locks the
// infra-sql-table-name-source rule: every table identifier comes from a
// registry/entityid constant (spot-checked by presence), and no education
// noun leaks into the statement (the code-generic C1 grep gate).
func TestOutcomeCompletionSQL_TableNamesFromEntityID(t *testing.T) {
	stmt, _ := ocStaffSQL(t)
	for _, tbl := range []string{
		entityid.Job, entityid.JobTemplate, entityid.JobPhase, entityid.JobTask,
		entityid.TemplateTaskCriteria, entityid.TaskOutcome, entityid.JobCategory,
		entityid.SubscriptionGroup, entityid.SubscriptionGroupMember,
		entityid.SubscriptionGroupProductPlanStaff, entityid.ProductPlan,
		entityid.ProductPlanStaff, entityid.SubscriptionSeat,
	} {
		if !strings.Contains(stmt, " "+tbl+" ") && !strings.Contains(stmt, " "+tbl+"\n") {
			t.Errorf("statement must reference entityid table %q", tbl)
		}
	}
	for _, noun := range []string{"student", "teacher", "grade", "grading", "class", "section", "semester", "school"} {
		if strings.Contains(strings.ToLower(stmt), noun) {
			t.Errorf("education noun %q leaked into the generic statement", noun)
		}
	}
}

// TestOutcomeCompletionSQL_CurrentWindowAndSynthesizedExclusion — the job set
// is membership-anchored to the CURRENT enrollment window and synthesized
// phase rows are excluded EXPLICITLY (F5 ride-along).
func TestOutcomeCompletionSQL_CurrentWindowAndSynthesizedExclusion(t *testing.T) {
	stmt, _ := buildOutcomeCompletionSummarySQL("ws-1", nil)
	for _, tok := range []string{
		"cg.status = 'current'",
		"cm.subscription_id = j.origin_id",
		"cm.client_id = j.client_id",
		"j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'",
		"NOT jp.is_synthesized",
	} {
		if !strings.Contains(stmt, tok) {
			t.Errorf("statement missing current-window/synthesized token %q", tok)
		}
	}
}

// TestOutcomeCompletionSQL_CellGrainAndDisjointBuckets — the expected cell is
// active job_task × active template_task_criteria, recorded matches the cell's
// criteria version, and the six approval buckets carry the disjointness
// predicates (returned = IN_PROGRESS + returned_at; not_started = IN_PROGRESS
// with no submit, no return, no recorded cell).
func TestOutcomeCompletionSQL_CellGrainAndDisjointBuckets(t *testing.T) {
	stmt, _ := buildOutcomeCompletionSummarySQL("ws-1", nil)
	for _, tok := range []string{
		"ttc.job_template_task_id = tk.template_task_id AND ttc.active",
		"tox.criteria_version_id = ttc.outcome_criteria_id",
		"AND returned_at IS NOT NULL",
		"AND returned_at IS NULL AND submitted_at IS NULL",
		"AND NOT has_recorded",
		"(submitted_at IS NOT NULL OR has_recorded)",
	} {
		if !strings.Contains(stmt, tok) {
			t.Errorf("statement missing cell/bucket token %q", tok)
		}
	}
}

// TestOutcomeCompletionSQL_FixedOrderByAndPeriodDimension — the ORDER BY is a
// fixed literal (no interpolation surface) and phase_order (the A2 period
// dimension) is grouped and projected.
func TestOutcomeCompletionSQL_FixedOrderByAndPeriodDimension(t *testing.T) {
	stmt, _ := buildOutcomeCompletionSummarySQL("ws-1", nil)
	if !strings.Contains(stmt, "ORDER BY cr.category_sort ASC, cr.category_id ASC NULLS LAST, ca.phase_order ASC NULLS LAST") {
		t.Errorf("fixed ORDER BY missing or altered")
	}
	if !strings.Contains(stmt, "GROUP BY category_id, phase_order") {
		t.Errorf("A2 period dimension grouping missing")
	}
}

// TestOutcomeCompletionSQL_WorkspacePredicateOnScopedTables — every
// workspace-carrying table in the base query binds $1.
func TestOutcomeCompletionSQL_WorkspacePredicateOnScopedTables(t *testing.T) {
	stmt, args := buildOutcomeCompletionSummarySQL("ws-1", nil)
	if len(args) != 1 || args[0] != "ws-1" {
		t.Fatalf("unscoped statement must bind exactly the workspace, got %v", args)
	}
	for _, tok := range []string{
		"j.workspace_id = $1",
		"jt.workspace_id = $1 AND jt.active",
		"cg.workspace_id = $1",
		"cm.workspace_id = $1",
		"cat.active AND cat.workspace_id = $1",
	} {
		if !strings.Contains(stmt, tok) {
			t.Errorf("statement missing workspace bound %q", tok)
		}
	}
}

// TestOutcomeCompletionScope_FailClosedKinds — the ctx-derived scope seam:
// no identity, the unresolved kind-0 sentinel, a client kind, and a staff
// session with an empty principal id ALL collapse to the always-false
// predicate (zero rows); operator kinds leave the query unchanged.
func TestOutcomeCompletionScope_FailClosedKinds(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		clause string
		nArgs  int
	}{
		{"no identity", context.Background(), " AND 1=0", 0},
		{"kind 0 sentinel", ocCtx(0, "", "ws-1"), " AND 1=0", 0},
		{"client kind", ocCtx(3, "client-1", "ws-1"), " AND 1=0", 0},
		{"staff empty id", ocCtx(7, "", "ws-1"), " AND 1=0", 0},
		{"operator owner", ocCtx(1, "own-1", "ws-1"), "", 0},
		{"operator staff", ocCtx(2, "op-1", "ws-1"), "", 0},
	}
	for _, tc := range cases {
		clause, args := outcomeCompletionCellScope(tc.ctx, 2)
		if clause != tc.clause || len(args) != tc.nArgs {
			t.Errorf("%s: want (%q, %d args), got (%q, %d args)", tc.name, tc.clause, tc.nArgs, clause, len(args))
		}
	}
}

// TestOutcomeCompletionScope_StaffParameterized — the staff predicate is fully
// parameterized: the session staff id and workspace ride ONLY in args, never
// in the SQL text.
func TestOutcomeCompletionScope_StaffParameterized(t *testing.T) {
	stmt, args := ocStaffSQL(t)
	if len(args) != 3 {
		t.Fatalf("staff-scoped statement must bind (workspace, staff, session-workspace), got %v", args)
	}
	if args[1] != "staff-rot-1" || args[2] != "ws-session" {
		t.Errorf("scope args must be session staff id + session workspace, got %v", args[1:])
	}
	for _, leak := range []string{"staff-rot-1", "ws-session"} {
		if strings.Contains(stmt, leak) {
			t.Errorf("bind value %q leaked into SQL text", leak)
		}
	}
	if !strings.Contains(stmt, "tk.assigned_to = $2") {
		t.Errorf("task-grain tier must bind the staff placeholder")
	}
	if !strings.Contains(stmt, "e.workspace_id = $3") {
		t.Errorf("class-edge tier must bind the SESSION workspace placeholder")
	}
}

// TestOutcomeCompletionScope_MostSpecificWins — the A1 amendment, structurally
// asserted against the storybook-findings F2 worked example (the Design
// Semester-2 rotation teacher must NOT count Semester-1 strand cells):
//
//  1. task grain outranks everything (tk.assigned_to = $s first, and lower
//     tiers run only when the task is unassigned);
//  2. the staff's own phase-scoped edge is restricted to ITS phase
//     (e.job_template_phase_id = jp.template_phase_id — the phase-blind
//     class-edge tier of the mirrored union is NOT reproduced for scoped
//     edges);
//  3. the job-grain tiers are guarded by the sibling NOT EXISTS, so a
//     phase covered by ANY live phase-scoped edge is claimable only through
//     tier 2 — sibling-semester teachers' job-grain edges cannot absorb it;
//  4. the phase-BLANK edge tier explicitly requires f14 IS NULL.
func TestOutcomeCompletionScope_MostSpecificWins(t *testing.T) {
	ctx := ocCtx(7, "staff-rot-1", "ws-session")
	clause, _ := outcomeCompletionCellScope(ctx, 2)

	// (1) task grain first + unassigned guard for lower tiers.
	idxTask := strings.Index(clause, "tk.assigned_to = $2")
	idxUnassigned := strings.Index(clause, "NULLIF(tk.assigned_to, '') IS NULL")
	if idxTask < 0 || idxUnassigned < 0 || idxTask > idxUnassigned {
		t.Fatalf("task-grain tier must lead and gate the lower tiers")
	}

	// (2) the phase-scoped own-edge tier restricts to the phase.
	if !strings.Contains(clause, "e.job_template_phase_id = jp.template_phase_id") {
		t.Errorf("phase-scoped edge must restrict to job_phase.template_phase_id (A1)")
	}
	if !strings.Contains(clause, "COALESCE(pps.staff_id, e.staff_id) = $2") {
		t.Errorf("phase-scoped edge must resolve staff v2-natively (COALESCE f13→f10)")
	}

	// (3) sibling guard: a NOT EXISTS over phase-scoped edges WITHOUT a staff
	// match, ANDed before the job-grain tiers.
	idxGuard := strings.Index(clause, "NOT EXISTS (SELECT 1 FROM "+entityid.SubscriptionGroupProductPlanStaff+" e2")
	if idxGuard < 0 {
		t.Fatalf("sibling phase-scope NOT EXISTS guard missing")
	}
	guard := clause[idxGuard:]
	end := strings.Index(guard, ") AND (")
	if end < 0 {
		t.Fatalf("sibling guard must gate the job-grain tier group")
	}
	if strings.Contains(guard[:end], "= $2") {
		t.Errorf("the sibling guard must match edges of ANY staff (no staff bind inside it)")
	}
	if !strings.Contains(guard[:end], "e2.job_template_phase_id = jp.template_phase_id") {
		t.Errorf("the sibling guard must key on the phase's template_phase_id")
	}
	for _, jobGrainTok := range []string{
		"txo.recorded_by = $2 OR txo.reviewed_by = $2", // delivery axis
		"ss.staff_id = $2",                 // seat tier
		"e3.job_template_phase_id IS NULL", // phase-blank class edge (4)
	} {
		if idx := strings.Index(clause, jobGrainTok); idx < idxGuard {
			t.Errorf("job-grain tier %q must sit AFTER (inside) the sibling guard", jobGrainTok)
		}
	}

	// (4) eligibility-liveness guard on every class-edge variant (M5-G5).
	for _, tok := range []string{
		"(e.product_plan_staff_id IS NULL OR pps.active)",
		"(e2.product_plan_staff_id IS NULL OR pps2.active)",
		"(e3.product_plan_staff_id IS NULL OR pps3.active)",
	} {
		if !strings.Contains(clause, tok) {
			t.Errorf("class-edge eligibility-live guard missing: %q", tok)
		}
	}
}

// TestOutcomeCompletionSQL_BalancedParens — the assembled staff-scoped
// statement is parenthesis-balanced (the live-psql PREPARE caught an
// unbalanced scope group during the build; this locks the invariant).
func TestOutcomeCompletionSQL_BalancedParens(t *testing.T) {
	stmt, _ := ocStaffSQL(t)
	depth := 0
	for _, r := range stmt {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth < 0 {
			t.Fatalf("close-paren before open in statement")
		}
	}
	if depth != 0 {
		t.Fatalf("unbalanced parens: depth %d at end of statement", depth)
	}
}

// TestOutcomeCompletionScope_OperatorGrainMatchesUnscoped — kinds 1/2 produce
// the byte-identical unscoped statement (workspace grain).
func TestOutcomeCompletionScope_OperatorGrainMatchesUnscoped(t *testing.T) {
	base, baseArgs := buildOutcomeCompletionSummarySQL("ws-1", nil)
	op, opArgs := buildOutcomeCompletionSummarySQL("ws-1", func(start int) (string, []any) {
		return outcomeCompletionCellScope(ocCtx(2, "op-1", "ws-1"), start)
	})
	if base != op {
		t.Errorf("operator-kind statement must be byte-identical to the unscoped statement")
	}
	if len(baseArgs) != 1 || len(opArgs) != 1 {
		t.Errorf("operator grain binds only the workspace, got %v / %v", baseArgs, opArgs)
	}
}
