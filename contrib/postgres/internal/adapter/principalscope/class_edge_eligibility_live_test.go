//go:build postgresql

package principalscope

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Coverage for audit gap M5-G5 (docs/plan/20260724-section-assignment-merged/
// coverage-audit-20260725.md §3): the class-edge (sgpps) tier's
// "LEFT JOIN product_plan_staff pps" carried NO pps.active predicate, so a
// REVOKED eligibility row never retracted reachability — the staff kept reaching
// the cohort's jobs and clients until the assignment EDGE itself was deactivated.
//
// The fix is the production constant classEdgeEligibilityLive, appended at all
// four class-edge sites (reachableClientUnion, reachableJobUnion,
// StaffReachableClientExistsSQL, StaffReachableJobExistsSQL). The tests below
// use that constant BY REFERENCE and splice it into the exact WHERE shape
// production builds, so a change to the predicate string is exercised here
// without editing the test.
//
// Two tests, two jobs:
//
//   - TestClassEdgeEligibilityLive_TruthTable executes the real predicate against
//     synthetic VALUES relations (no application table is read or written),
//     covering every (link, eligibility-state, legacy-value) combination —
//     including the NAIVE-FIX TRAP, the case that proves why "AND pps.active"
//     belongs in the WHERE and not in the JOIN condition.
//   - TestClassEdgeEligibilityLive_LiveSubsetAndNoOp runs the pre-fix and
//     post-fix class-edge branches side by side over EVERY (staff, workspace)
//     pair on the live database and proves the fix is a strict retraction, and a
//     no-op while no eligibility is revoked.
//
// Both are gated on TEST_DATABASE_URL via openClassEdgeV2LiveDB (this package's
// sibling idiom) and skip cleanly when no database is reachable.

// eligibilityTargetStaff is the staff.id the truth-table binds as $1 — the
// principal whose reachability is under test.
const eligibilityTargetStaff = "staff-T"

// eligibilityOtherStaff is a DIFFERENT staff.id, used to make cases discriminating
// (a case that resolves to this value must be denied, whatever the predicate).
const eligibilityOtherStaff = "staff-OTHER"

// classEdgeEligibilityCase is one synthetic (sgpps edge, product_plan_staff)
// pair. It carries the expected verdict under BOTH the pre-fix and the post-fix
// predicate, so the table documents the behaviour change row by row instead of
// only asserting the new state.
type classEdgeEligibilityCase struct {
	name string
	// link is e.product_plan_staff_id (f13). "" means SQL NULL — the unlinked,
	// pre-M3 migration shape.
	link string
	// legacyStaffID is the edge's own staff_id (f10), still dual-written until M7.
	legacyStaffID string
	// edgeActive is e.active.
	edgeActive bool
	// ppsExists=false with a non-empty link models a dangling/invisible link.
	ppsExists  bool
	ppsStaffID string
	ppsActive  bool

	wantPreFix  bool // did the pre-fix predicate grant this row?
	wantPostFix bool // does the shipped predicate grant it?
	why         string
}

func classEdgeEligibilityCases() []classEdgeEligibilityCase {
	return []classEdgeEligibilityCase{
		{
			name: "no_link__legacy_matches", link: "", legacyStaffID: eligibilityTargetStaff,
			edgeActive: true,
			wantPreFix: true, wantPostFix: true,
			why: "MIGRATION FALLBACK: the edge has no f13 link, so the LEFT JOIN yields no pps row and COALESCE must fall back to legacy f10. This is how every row behaved before M3 and how any future unlinked row behaves; the fix must not touch it.",
		},
		{
			name: "no_link__legacy_differs", link: "", legacyStaffID: eligibilityOtherStaff,
			edgeActive: true,
			wantPreFix: false, wantPostFix: false,
			why: "control: an unlinked edge belonging to a DIFFERENT staff must not resolve to the target under either predicate (proves the table is not vacuously granting).",
		},
		{
			name: "link_active__pps_matches", link: "pps-1", legacyStaffID: eligibilityTargetStaff,
			edgeActive: true, ppsExists: true, ppsStaffID: eligibilityTargetStaff, ppsActive: true,
			wantPreFix: true, wantPostFix: true,
			why: "the post-M3 live shape (379/379 edges linked, dual-write keeps f10 in step): a live eligibility grants, unchanged.",
		},
		{
			name: "link_active__pps_authoritative_over_legacy", link: "pps-2", legacyStaffID: eligibilityOtherStaff,
			edgeActive: true, ppsExists: true, ppsStaffID: eligibilityTargetStaff, ppsActive: true,
			wantPreFix: true, wantPostFix: true,
			why: "f13 is PREFERRED, not merely a null-rescue: the linked pps staff wins over a disagreeing legacy f10 value.",
		},
		{
			name: "link_active__pps_differs__legacy_matches", link: "pps-3", legacyStaffID: eligibilityTargetStaff,
			edgeActive: true, ppsExists: true, ppsStaffID: eligibilityOtherStaff, ppsActive: true,
			wantPreFix: false, wantPostFix: false,
			why: "the other direction of the same rule: once linked, a stale legacy f10 pointing at the target must NOT rescue it — COALESCE resolves to the pps staff and the target is denied.",
		},
		{
			name: "link_revoked__pps_matches", link: "pps-4", legacyStaffID: eligibilityOtherStaff,
			edgeActive: true, ppsExists: true, ppsStaffID: eligibilityTargetStaff, ppsActive: false,
			wantPreFix: true, wantPostFix: false,
			why: "M5-G5, the defect itself: revoking the eligibility retracted nothing, because pps.active played no part in the query.",
		},
		{
			name: "link_revoked__legacy_matches", link: "pps-5", legacyStaffID: eligibilityTargetStaff,
			edgeActive: true, ppsExists: true, ppsStaffID: eligibilityTargetStaff, ppsActive: false,
			wantPreFix: true, wantPostFix: false,
			why: "NAIVE-FIX REGRESSION GUARD: the eligibility is revoked but legacy f10 still names the same staff. Moving the liveness check into the JOIN condition would null the pps row out and let COALESCE fall through to f10, re-granting the revoked staff. Only a WHERE predicate keyed on e.product_plan_staff_id denies this row.",
		},
		{
			name: "link_dangling__legacy_matches", link: "pps-absent", legacyStaffID: eligibilityTargetStaff,
			edgeActive: true, ppsExists: false,
			wantPreFix: true, wantPostFix: false,
			why: "FAIL-CLOSED: the edge claims an eligibility row that cannot be seen. pps.active is NULL, the predicate is NULL (not TRUE), the row is dropped — consistent with every other tier here.",
		},
		{
			name: "inactive_edge__link_active", link: "pps-6", legacyStaffID: eligibilityTargetStaff,
			edgeActive: false, ppsExists: true, ppsStaffID: eligibilityTargetStaff, ppsActive: true,
			wantPreFix: false, wantPostFix: false,
			why: "control: the pre-existing e.active gate is independent of the new predicate and still denies a deactivated assignment edge.",
		},
	}
}

// sqlTextLit renders a Go string as a typed SQL text literal, mapping "" to NULL
// (the shape a nullable f13 / dangling link takes).
func sqlTextLit(s string) string {
	if s == "" {
		return "NULL::text"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'::text"
}

// classEdgeEligibilityFixture builds the two synthetic relations the class-edge
// branch joins — aliased "e" and "pps", the SAME aliases production uses, which
// is what lets the production predicate string splice in verbatim.
func classEdgeEligibilityFixture(cases []classEdgeEligibilityCase) string {
	edgeRows := make([]string, 0, len(cases))
	ppsRows := make([]string, 0, len(cases))
	for _, c := range cases {
		edgeRows = append(edgeRows, fmt.Sprintf("(%s, %s, %s, %t)",
			sqlTextLit(c.name), sqlTextLit(c.link), sqlTextLit(c.legacyStaffID), c.edgeActive))
		if c.ppsExists {
			ppsRows = append(ppsRows, fmt.Sprintf("(%s, %s, %t)",
				sqlTextLit(c.link), sqlTextLit(c.ppsStaffID), c.ppsActive))
		}
	}
	return "WITH e(name, product_plan_staff_id, staff_id, active) AS (VALUES\n  " +
		strings.Join(edgeRows, ",\n  ") + "\n),\n" +
		"pps(id, staff_id, active) AS (VALUES\n  " +
		strings.Join(ppsRows, ",\n  ") + "\n)\n"
}

// grantedCaseNames executes one query form and returns the set of case names it
// granted for eligibilityTargetStaff.
func grantedCaseNames(t *testing.T, db *sql.DB, stmt string) map[string]bool {
	t.Helper()
	rows, err := db.Query(stmt, eligibilityTargetStaff)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, stmt)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestClassEdgeEligibilityLive_TruthTable executes the REAL production predicate
// (classEdgeEligibilityLive, referenced not retyped) spliced into the REAL WHERE
// shape the four class-edge sites build, against synthetic VALUES relations. It
// asserts the full truth table in both directions: what the pre-fix query granted
// and what the shipped query grants.
func TestClassEdgeEligibilityLive_TruthTable(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()

	cases := classEdgeEligibilityCases()
	fixture := classEdgeEligibilityFixture(cases)

	// The WHERE below is production's class-edge WHERE with the three
	// graph-walk joins (subscription_group_member / product_plan / job) and the
	// workspace bind removed — none of them touch staff resolution, and all four
	// production sites share exactly this staff/eligibility/active core.
	const selectFrom = "SELECT e.name FROM e LEFT JOIN pps ON pps.id = e.product_plan_staff_id\n"
	const staffPredicate = "WHERE COALESCE(pps.staff_id, e.staff_id) = $1"

	preFix := fixture + selectFrom + staffPredicate + " AND e.active"
	postFix := fixture + selectFrom + staffPredicate + classEdgeEligibilityLive + " AND e.active"

	gotPre := grantedCaseNames(t, db, preFix)
	gotPost := grantedCaseNames(t, db, postFix)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if gotPre[c.name] != c.wantPreFix {
				t.Errorf("PRE-FIX predicate granted=%t, want %t — the pre-fix baseline this case documents is wrong, so the behaviour-change claim below is unproven.\nrationale: %s",
					gotPre[c.name], c.wantPreFix, c.why)
			}
			if gotPost[c.name] != c.wantPostFix {
				t.Errorf("SHIPPED predicate %q granted=%t, want %t\nrationale: %s",
					strings.TrimSpace(classEdgeEligibilityLive), gotPost[c.name], c.wantPostFix, c.why)
			}
		})
	}

	// The fix must only ever REMOVE grants: anything the shipped predicate grants
	// must also have been granted before it. A post-fix-only grant would mean the
	// predicate widened reachability, which is never a correct outcome here.
	for name := range gotPost {
		if !gotPre[name] {
			t.Errorf("case %q is granted by the SHIPPED predicate but was denied pre-fix — the eligibility gate must be a strict retraction, never a widening", name)
		}
	}
}

// TestClassEdgeEligibilityLive_NaiveJoinFixIsInsufficient is the explicit
// counter-example for the fix that looks obvious and is wrong: moving the
// liveness check into the LEFT JOIN condition ("... ON pps.id =
// e.product_plan_staff_id AND pps.active") instead of the WHERE.
//
// Under that form a revoked pps row simply fails to match; the LEFT JOIN nulls it
// out; COALESCE(pps.staff_id, e.staff_id) then falls through to the legacy f10
// column, which the M2 dual-write still populates — so the revoked staff is
// granted anyway and the defect survives untouched.
func TestClassEdgeEligibilityLive_NaiveJoinFixIsInsufficient(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()

	cases := classEdgeEligibilityCases()
	fixture := classEdgeEligibilityFixture(cases)

	naive := fixture +
		"SELECT e.name FROM e LEFT JOIN pps ON pps.id = e.product_plan_staff_id AND pps.active\n" +
		"WHERE COALESCE(pps.staff_id, e.staff_id) = $1 AND e.active"

	gotNaive := grantedCaseNames(t, db, naive)

	const trap = "link_revoked__legacy_matches"
	if !gotNaive[trap] {
		t.Fatalf("case %q was DENIED by the naive join-condition form — this test's whole premise (that the naive fix leaks) no longer holds; re-derive the fixture before trusting the shipped predicate's justification", trap)
	}
	t.Logf("confirmed: the naive join-condition form still grants %q (revoked eligibility rescued by the legacy f10 fallback)", trap)

	// And it is not merely incomplete, it is INCONSISTENT: the same revocation is
	// honoured or ignored depending on whether legacy f10 happens to name the same
	// staff — which is exactly the accident the shipped WHERE predicate removes.
	const honoured = "link_revoked__pps_matches"
	if gotNaive[honoured] {
		t.Errorf("case %q: expected the naive form to deny here (pps nulls out, legacy names a different staff); it granted", honoured)
	}
}

// countRevokedLinkedEdges reports how many ACTIVE sgpps edges carry an f13 link
// whose product_plan_staff row is inactive or unreachable — i.e. how many live
// rows the new predicate can actually retract. All 108 product_plan_staff rows
// were active on 2026-07-25, so this is 0 and the fix is a live no-op.
func countRevokedLinkedEdges(t *testing.T, db *sql.DB) int {
	t.Helper()
	q := "SELECT count(*) FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" LEFT JOIN " + entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id" +
		" WHERE e.active AND e.product_plan_staff_id IS NOT NULL" +
		" AND (pps.id IS NULL OR NOT pps.active)"
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("count revoked linked edges: %v", err)
	}
	return n
}

// classEdgeBranchIDs runs one form of the class-edge branch over EVERY (staff,
// workspace) pair at once — no sampling — and returns the resulting
// (staff, workspace, id) triples. grainCol selects the client_id or job.id grain,
// matching reachableClientUnion / reachableJobUnion respectively.
func classEdgeBranchIDs(t *testing.T, db *sql.DB, grainCol, predicate string) map[string]bool {
	t.Helper()
	q := "SELECT DISTINCT COALESCE(pps.staff_id, e.staff_id) AS staff_id, e.workspace_id, " + grainCol + " AS grain" +
		" FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" LEFT JOIN " + entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id" +
		" WHERE e.active" + predicate
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, q)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var staffID, wsID, grain string
		if err := rows.Scan(&staffID, &wsID, &grain); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[staffID+"|"+wsID+"|"+grain] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestClassEdgeEligibilityLive_LiveSubsetAndNoOp is the comparative live-data
// test: it runs the PRE-FIX and POST-FIX class-edge branches side by side over
// every (staff, workspace) pair on the real database — read-only — at both the
// client_id and job.id grain, and asserts:
//
//   - the post-fix set is always a SUBSET of the pre-fix set (a permanent safety
//     property: this predicate may only retract reachability, never add it), and
//   - while no eligibility is revoked, the two sets are IDENTICAL — the empirical
//     no-regression proof for today's data (all 108 product_plan_staff rows
//     active, 379/379 edges linked, 0 dangling links as of 2026-07-25).
//
// The equality half is asserted only when countRevokedLinkedEdges reports 0.
// Asserting it unconditionally would be dishonest: the day an eligibility IS
// revoked, divergence is the CORRECT outcome and the whole point of the fix.
func TestClassEdgeEligibilityLive_LiveSubsetAndNoOp(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()

	revoked := countRevokedLinkedEdges(t, db)
	t.Logf("live active sgpps edges with a revoked/unreachable eligibility link: %d", revoked)

	for _, grain := range []struct{ label, col string }{
		{"client_id", "jce.client_id"},
		{"job_id", "jce.id"},
	} {
		t.Run(grain.label, func(t *testing.T) {
			preFix := classEdgeBranchIDs(t, db, grain.col, "")
			postFix := classEdgeBranchIDs(t, db, grain.col, classEdgeEligibilityLive)

			if len(preFix) == 0 {
				t.Skip("class-edge branch reaches zero rows on this database — nothing to compare")
			}

			for k := range postFix {
				if !preFix[k] {
					t.Errorf("post-fix reaches %q which the pre-fix query did not — the eligibility gate must only retract reachability", k)
				}
			}

			if revoked == 0 {
				if len(postFix) != len(preFix) {
					var lost []string
					for k := range preFix {
						if !postFix[k] {
							lost = append(lost, k)
						}
					}
					t.Fatalf("no live eligibility is revoked, so the predicate must be a no-op, but %d of %d reachable rows were dropped (first few: %v)",
						len(preFix)-len(postFix), len(preFix), lost[:min(5, len(lost))])
				}
				t.Logf("no-op confirmed: %d reachable %s rows, identical before and after", len(preFix), grain.label)
				return
			}
			t.Logf("%d revoked-eligibility edges exist; retraction is expected: %d → %d reachable %s rows",
				revoked, len(preFix), len(postFix), grain.label)
		})
	}
}
