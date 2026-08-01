//go:build postgresql

package operation

import (
	"strings"
	"testing"
)

// Shape suite for the D7 submit-ownership gate under the 2026-07-26 COALESCE
// model (plan 20260726-grade-cell-edit-guard §3). These tests pin the SQL
// structure without a database; the behavioral ownership matrix lives in
// job_phase_ownership_integration_test.go (TEST_DATABASE_URL-gated).

// TestClassEdgeOwnedSQLShape locks the submit-side class-edge fragment to the
// locked model's term set. Every load-bearing term of the cells query's
// classEdgeExpr (outcome_matrix_query.go — the cell-edit guard) must appear here
// too: the submit gate and the cell-edit guard must never disagree about who
// owns an unassigned task.
func TestClassEdgeOwnedSQLShape(t *testing.T) {
	sql := classEdgeOwnedSQL(4, 3)
	for _, frag := range []string{
		// membership: the client's ACTIVE, workspace-bound section membership
		"m.client_id = j.client_id",
		"m.active AND m.workspace_id = $3",
		// completed/draft sections grant nothing (the completed-AY leak gate)
		"sg.status = 'current'",
		// class: active + workspace-bound
		"c.active AND c.workspace_id = $3",
		// product match — a legacy NULL output_product_id can never satisfy this,
		// so legacy tasks stay ownable ONLY via explicit assigned_to
		"pp.product_id = j.output_product_id",
		// the edge: active, PRIMARY only, the acting facet, workspace-bound
		"e.active AND e.role = 'primary'",
		"e.staff_id = $4",
		"e.workspace_id = $3",
		// f14: class-wide (NULL) or phase-scoped by ORDER, not id
		"e.job_template_phase_id IS NULL",
		"ep.id = e.job_template_phase_id",
		"jpp.id = jp.template_phase_id",
		"ep.phase_order = jpp.phase_order",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("class-edge fragment missing %q:\n%s", frag, sql)
		}
	}
}

// TestClassEdgeOwnedSQL_PlaceholdersAreCallerSupplied proves both placeholder
// indexes are honored — the ownership probe binds facet at $4 / workspace at $3,
// but a future caller with a different arg layout must not silently compare
// against the wrong bind.
func TestClassEdgeOwnedSQL_PlaceholdersAreCallerSupplied(t *testing.T) {
	sql := classEdgeOwnedSQL(7, 9)
	for _, frag := range []string{
		"e.staff_id = $7",
		"e.workspace_id = $9",
		"c.workspace_id = $9",
		"m.workspace_id = $9",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("re-numbered fragment missing %q", frag)
		}
	}
	for _, stale := range []string{"$3", "$4"} {
		if strings.Contains(sql, stale) {
			t.Errorf("re-numbered fragment still carries stale placeholder %q", stale)
		}
	}
}

// TestTaskUnownedProbeSQLShape locks the COALESCE two-leg ownership structure:
// override leg (assigned_to present → it ALONE decides) OR edge leg (assigned_to
// absent → the class edge governs), wrapped in NOT(...) to select unowned rows.
// The COALESCE() forms keep the predicate two-valued — a raw `assigned_to = $4`
// against NULL would collapse the NOT(...) to SQL NULL and silently drop the row
// from the unowned probe (fail OPEN).
func TestTaskUnownedProbeSQLShape(t *testing.T) {
	sql := taskUnownedProbeSQL("")
	for _, frag := range []string{
		// sheet scope + trusted workspace bind, active ancestry
		"jp.template_phase_id = $2",
		"j.job_template_id = $1",
		"j.workspace_id = $3",
		"jp.active = true",
		"jt.active = true",
		// the two ownership legs, two-valued
		"COALESCE(jt.assigned_to, '') <> '' AND jt.assigned_to = $4",
		"COALESCE(jt.assigned_to, '') = '' AND EXISTS (",
		// fail-closed wrapper: unowned = NOT(override OR edge)
		"AND NOT (",
		// the edge leg is the mirrored class-edge chain
		"sg.status = 'current'",
		"e.role = 'primary'",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("unowned probe missing %q:\n%s", frag, sql)
		}
	}
}

// TestTaskUnownedProbeSQL_GroupNarrow proves the group-narrow predicate is
// spliced into the probe (placeholders $5 group / $3 workspace — $4 is the
// facet) and that an absent group changes nothing.
func TestTaskUnownedProbeSQL_GroupNarrow(t *testing.T) {
	narrow, args := groupNarrowPredicate("grp-1", 5, 3)
	if len(args) != 1 || args[0] != "grp-1" {
		t.Fatalf("unexpected narrow args: %v", args)
	}
	sql := taskUnownedProbeSQL(narrow)
	for _, frag := range []string{
		"sgm_g.subscription_group_id = $5",
		"sgm_g.workspace_id = $3",
		"sgm_g.subscription_id = j.origin_id",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("narrowed probe missing %q", frag)
		}
	}
	if strings.Contains(taskUnownedProbeSQL(""), "sgm_g") {
		t.Error("probe gained a group join with no group supplied")
	}
}
