//go:build postgresql

package operation

import (
	"strings"
	"testing"
)

// TestCodedHistoricalSQL_ShapeInvariants locks the load-bearing differences (and
// preserved guarantees) of the Q6 historical coded-cell reader relative to the
// live one, so a future edit cannot silently re-introduce the past-card blank or
// weaken tenant/criterion/outcome scoping.
func TestCodedHistoricalSQL_ShapeInvariants(t *testing.T) {
	live := codedTaskOutcomeValuesByJobSQL(2)
	hist := codedTaskOutcomeValuesByJobHistoricalSQL(2)

	// (1) Tenant isolation is UNCHANGED — the workspace bind must still be present.
	for _, sub := range []string{"j.workspace_id = $3", "j.id IN ($1, $2)"} {
		if !strings.Contains(hist, sub) {
			t.Fatalf("historical SQL missing tenant/allowlist predicate %q", sub)
		}
	}

	// (2) The ancestry active predicates the live path carries MUST be gone in the
	//     historical path (that is exactly why a past card resolved nothing).
	for _, pred := range []string{
		"jp.active = true",
		"jtp.active = true",
		"jt.active = true",
		"jtt.active = true",
		"ttc.active = true",
		"AND j.active = true",
	} {
		if !strings.Contains(live, pred) {
			t.Fatalf("precondition failed: live SQL no longer contains %q (test is stale)", pred)
		}
		if strings.Contains(hist, pred) {
			t.Fatalf("historical SQL must DROP ancestry active predicate %q (past cards would stay blank)", pred)
		}
	}

	// (3) Criterion codes stay ACTIVE-only and a single latest active outcome per
	//     cell is preserved — both guards must survive in the historical path.
	for _, keep := range []string{
		"oc.active = true",
		"o.active = true",
		"SELECT DISTINCT ON (jt.id, ttc.outcome_criteria_id)",
		"ORDER BY jt.id, ttc.outcome_criteria_id, o.recorded_date DESC NULLS LAST, o.id DESC",
	} {
		if !strings.Contains(hist, keep) {
			t.Fatalf("historical SQL must KEEP guarantee %q", keep)
		}
	}

	// (4) Codes come from PK-equality joins to the referenced ancestry (no natural-
	//     key twin selection), so an inactive uncoded twin cannot shadow the owner.
	for _, join := range []string{
		"jtp.id = jp.template_phase_id",
		"jtt.id = jt.template_task_id",
		"oc.id = ttc.outcome_criteria_id",
	} {
		if !strings.Contains(hist, join) {
			t.Fatalf("historical SQL missing id-equality ancestry join %q (twin-shadow safety)", join)
		}
	}
}
