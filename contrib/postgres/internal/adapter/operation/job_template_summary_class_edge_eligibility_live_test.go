//go:build postgresql

package operation

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Courses-fold half of audit gap M5-G5 (docs/plan/20260724-section-assignment-merged/
// coverage-audit-20260725.md §3, which names this file's dd CTE branch (b)
// alongside principalscope.go as the two M5 consumers). Branch (b) resolved the
// class's teacher-of-record as COALESCE(pps.staff_id, e.staff_id) over a LEFT
// JOIN on the f13 eligibility link, with no pps.active predicate — so revoking a
// teacher's eligibility left them attributed as the deliverer indefinitely.
//
// The fix is the production constant classEdgeEligibilityLivePredicate. The tests
// below reference that constant rather than retyping it, so the predicate cannot
// drift away from what is asserted. Companion:
// contrib/postgres/internal/adapter/principalscope/class_edge_eligibility_live_test.go
// (same predicate, the authorization consumer).

// openFoldLiveDB is openJTSLiveDB's alias-free entry point for these tests (same
// TEST_DATABASE_URL gate, same clean skip).
func openFoldLiveDB(t *testing.T) *sql.DB {
	t.Helper()
	return openJTSLiveDB(t)
}

const (
	foldTargetStaff = "staff-T"
	foldOtherStaff  = "staff-OTHER"
)

// foldEligibilityCase is one synthetic (sgpps edge, product_plan_staff) pair for
// the dd CTE's branch (b). wantPreFix/wantPostFix hold the staff.id the fold
// ATTRIBUTES as deliverer, or "" when the branch emits no row at all.
type foldEligibilityCase struct {
	name          string
	link          string // e.product_plan_staff_id (f13); "" => SQL NULL
	legacyStaffID string // e.staff_id (f10)
	role          string
	edgeActive    bool
	ppsExists     bool
	ppsStaffID    string
	ppsActive     bool

	wantPreFix  string
	wantPostFix string
	why         string
}

func foldEligibilityCases() []foldEligibilityCase {
	return []foldEligibilityCase{
		{
			name: "no_link", link: "", legacyStaffID: foldTargetStaff,
			role: "primary", edgeActive: true,
			wantPreFix: foldTargetStaff, wantPostFix: foldTargetStaff,
			why: "MIGRATION FALLBACK: unlinked edge (pre-M3 shape). The LEFT JOIN yields no pps row and legacy f10 is the correct attribution. Must not change.",
		},
		{
			name: "link_active", link: "pps-1", legacyStaffID: foldTargetStaff,
			role: "primary", edgeActive: true, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: true,
			wantPreFix: foldTargetStaff, wantPostFix: foldTargetStaff,
			why: "the post-M3 live shape: a live eligibility attributes its staff, unchanged.",
		},
		{
			name: "link_active_authoritative_over_legacy", link: "pps-2", legacyStaffID: foldOtherStaff,
			role: "primary", edgeActive: true, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: true,
			wantPreFix: foldTargetStaff, wantPostFix: foldTargetStaff,
			why: "f13 is PREFERRED over a disagreeing legacy f10 — the M5 cutover's own rule, unaffected by this fix.",
		},
		{
			name: "link_revoked_legacy_agrees", link: "pps-3", legacyStaffID: foldTargetStaff,
			role: "primary", edgeActive: true, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: false,
			wantPreFix: foldTargetStaff, wantPostFix: "",
			why: "NAIVE-FIX REGRESSION GUARD: eligibility revoked, legacy f10 still names the same teacher. Putting the liveness check in the JOIN condition would null the pps row and let COALESCE fall back to f10, re-attributing the revoked teacher. Only a WHERE predicate keyed on e.product_plan_staff_id suppresses this row.",
		},
		{
			name: "link_revoked_legacy_differs", link: "pps-4", legacyStaffID: foldOtherStaff,
			role: "primary", edgeActive: true, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: false,
			wantPreFix: foldTargetStaff, wantPostFix: "",
			why: "M5-G5 in the fold: a revoked eligibility kept naming its staff as teacher-of-record.",
		},
		{
			name: "link_dangling", link: "pps-absent", legacyStaffID: foldTargetStaff,
			role: "primary", edgeActive: true, ppsExists: false,
			wantPreFix: foldTargetStaff, wantPostFix: "",
			why: "FAIL-CLOSED: the edge claims an eligibility row that cannot be seen; pps.active is NULL, the predicate is NULL (not TRUE), the row is suppressed.",
		},
		{
			name: "secondary_role_link_active", link: "pps-5", legacyStaffID: foldTargetStaff,
			role: "secondary", edgeActive: true, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: true,
			wantPreFix: "", wantPostFix: "",
			why: "control: the pre-existing role='primary' (teacher-of-record, C11) gate is independent of the new predicate and still excludes co-teachers.",
		},
		{
			name: "inactive_edge_link_active", link: "pps-6", legacyStaffID: foldTargetStaff,
			role: "primary", edgeActive: false, ppsExists: true, ppsStaffID: foldTargetStaff, ppsActive: true,
			wantPreFix: "", wantPostFix: "",
			why: "control: the pre-existing e.active gate is independent of the new predicate.",
		},
	}
}

func foldTextLit(s string) string {
	if s == "" {
		return "NULL::text"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'::text"
}

// foldFixture builds synthetic "e", "pps" and "st" relations under the SAME
// aliases the dd CTE uses, which is what lets classEdgeEligibilityLivePredicate
// splice in verbatim.
func foldFixture(cases []foldEligibilityCase) string {
	edgeRows := make([]string, 0, len(cases))
	ppsRows := make([]string, 0, len(cases))
	for _, c := range cases {
		edgeRows = append(edgeRows, fmt.Sprintf("(%s, %s, %s, %s, %t)",
			foldTextLit(c.name), foldTextLit(c.link), foldTextLit(c.legacyStaffID),
			foldTextLit(c.role), c.edgeActive))
		if c.ppsExists {
			ppsRows = append(ppsRows, fmt.Sprintf("(%s, %s, %t)",
				foldTextLit(c.link), foldTextLit(c.ppsStaffID), c.ppsActive))
		}
	}
	return "WITH e(name, product_plan_staff_id, staff_id, role, active) AS (VALUES\n  " +
		strings.Join(edgeRows, ",\n  ") + "\n),\n" +
		"pps(id, staff_id, active) AS (VALUES\n  " +
		strings.Join(ppsRows, ",\n  ") + "\n),\n" +
		"st(id) AS (VALUES (" + foldTextLit(foldTargetStaff) + "), (" + foldTextLit(foldOtherStaff) + "))\n"
}

// foldAttribution runs one form of branch (b)'s staff resolution and returns
// case name → attributed staff.id (absent when the branch emits no row).
func foldAttribution(t *testing.T, db *sql.DB, stmt string) map[string]string {
	t.Helper()
	rows, err := db.Query(stmt)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, stmt)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, staffID string
		if err := rows.Scan(&name, &staffID); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[name] = staffID
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestJobTemplateSummaryFold_ClassEdgeEligibilityLive_TruthTable executes the
// REAL production predicate (classEdgeEligibilityLivePredicate, referenced not
// retyped) inside the dd CTE branch (b) staff-resolution shape — the LEFT JOIN to
// product_plan_staff, the INNER JOIN to staff on COALESCE(pps.staff_id,
// e.staff_id), and the e.active / role='primary' gates — against synthetic VALUES
// relations. No application table is read or written.
func TestJobTemplateSummaryFold_ClassEdgeEligibilityLive_TruthTable(t *testing.T) {
	db := openFoldLiveDB(t)
	defer db.Close()

	cases := foldEligibilityCases()
	fixture := foldFixture(cases)

	// Branch (b)'s resolution core, with the member/product_plan joins and the
	// correlated deterministic-primary pick removed — neither touches staff
	// resolution, and the pick is deliberately NOT gated by this predicate (see
	// the constant's doc comment on why it stays out of the e2 subquery).
	const selectFrom = "SELECT e.name, st.id FROM e\n" +
		" LEFT JOIN pps ON pps.id = e.product_plan_staff_id\n" +
		" JOIN st ON st.id = COALESCE(pps.staff_id, e.staff_id)\n"
	const baseWhere = " WHERE e.active AND e.role = 'primary'\n"

	gotPre := foldAttribution(t, db, fixture+selectFrom+baseWhere)
	gotPost := foldAttribution(t, db, fixture+selectFrom+baseWhere+" "+classEdgeEligibilityLivePredicate)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if gotPre[c.name] != c.wantPreFix {
				t.Errorf("PRE-FIX fold attributed %q, want %q — the documented pre-fix baseline is wrong, so the behaviour-change claim below is unproven.\nrationale: %s",
					gotPre[c.name], c.wantPreFix, c.why)
			}
			if gotPost[c.name] != c.wantPostFix {
				t.Errorf("SHIPPED fold predicate %q attributed %q, want %q\nrationale: %s",
					classEdgeEligibilityLivePredicate, gotPost[c.name], c.wantPostFix, c.why)
			}
		})
	}

	// The predicate may only suppress attributions, never invent or change one.
	for name, staffID := range gotPost {
		if gotPre[name] != staffID {
			t.Errorf("case %q: shipped fold attributes %q where the pre-fix fold attributed %q — this predicate must only SUPPRESS an attribution, never redirect it",
				name, staffID, gotPre[name])
		}
	}
}

// TestJobTemplateSummaryFold_NaiveJoinFixIsInsufficient is the courses-fold
// counter-example to the obvious-looking fix: moving the liveness check into the
// LEFT JOIN condition. A revoked pps row then fails to match, the LEFT JOIN nulls
// it out, and COALESCE falls through to the still-dual-written legacy f10 column
// — re-attributing the very teacher whose eligibility was revoked.
func TestJobTemplateSummaryFold_NaiveJoinFixIsInsufficient(t *testing.T) {
	db := openFoldLiveDB(t)
	defer db.Close()

	cases := foldEligibilityCases()
	naive := foldFixture(cases) +
		"SELECT e.name, st.id FROM e\n" +
		" LEFT JOIN pps ON pps.id = e.product_plan_staff_id AND pps.active\n" +
		" JOIN st ON st.id = COALESCE(pps.staff_id, e.staff_id)\n" +
		" WHERE e.active AND e.role = 'primary'"

	got := foldAttribution(t, db, naive)

	const trap = "link_revoked_legacy_agrees"
	if got[trap] != foldTargetStaff {
		t.Fatalf("case %q: the naive join-condition form attributed %q, expected it to still attribute %q — this test's premise (that the naive fix leaks through the legacy fallback) no longer holds; re-derive the fixture before trusting the shipped predicate's justification",
			trap, got[trap], foldTargetStaff)
	}
	t.Logf("confirmed: the naive join-condition form still attributes %q for %q (revoked eligibility rescued by legacy f10)", foldTargetStaff, trap)
}

// foldBranchBRows runs the dd CTE's branch (b) — the real join shape, over real
// tables, for EVERY workspace at once (no sampling) — in one of its two forms,
// and returns the emitted (subscription, client, product, staff, group) tuples.
func foldBranchBRows(t *testing.T, db *sql.DB, predicate string) map[string]bool {
	t.Helper()
	q := `SELECT DISTINCT m.subscription_id, m.client_id, pl.product_id, st.id, m.subscription_group_id
FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e
JOIN ` + entityid.SubscriptionGroupMember + ` m ON m.subscription_group_id = e.subscription_group_id AND m.active
JOIN ` + entityid.ProductPlan + ` pl ON pl.id = e.product_plan_id
LEFT JOIN ` + entityid.ProductPlanStaff + ` pps ON pps.id = e.product_plan_staff_id
JOIN ` + entityid.Staff + ` st ON st.id = COALESCE(pps.staff_id, e.staff_id) AND st.workspace_id = e.workspace_id
WHERE e.active AND e.role = 'primary' ` + predicate + `
  AND e.id = (
      SELECT e2.id FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e2
      WHERE e2.subscription_group_id = e.subscription_group_id
        AND e2.product_plan_id = e.product_plan_id
        AND e2.active AND e2.workspace_id = e.workspace_id AND e2.role = 'primary'
      ORDER BY e2.date_created DESC, e2.id DESC
      LIMIT 1
  )`
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, q)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var sub, client, product, staffID, group string
		if err := rows.Scan(&sub, &client, &product, &staffID, &group); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[strings.Join([]string{sub, client, product, staffID, group}, "|")] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestJobTemplateSummaryFold_ClassEdgeEligibilityLive_LiveSubsetAndNoOp runs the
// PRE-FIX and POST-FIX dd branch (b) side by side over every workspace on the
// real database — read-only — and asserts:
//
//   - the post-fix tuple set is always a SUBSET of the pre-fix set (permanent
//     safety property: this predicate may only suppress attribution), and
//   - while no eligibility is revoked, the two sets are IDENTICAL — the
//     empirical no-regression proof for today's data.
//
// The equality half is conditional on there being zero revoked links. Asserting
// it unconditionally would be dishonest: once an eligibility IS revoked,
// divergence is the correct and intended outcome.
//
// This matters more here than in principalscope because dd is INNER-joined to jj
// in the assembled statement, so suppressing branch (b) for a seatless section
// drops that section's jobs from the courses list rather than showing a stale
// teacher. Zero live rows are in that state today.
func TestJobTemplateSummaryFold_ClassEdgeEligibilityLive_LiveSubsetAndNoOp(t *testing.T) {
	db := openFoldLiveDB(t)
	defer db.Close()

	var revoked int
	revokedQ := "SELECT count(*) FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" LEFT JOIN " + entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id" +
		" WHERE e.active AND e.role = 'primary' AND e.product_plan_staff_id IS NOT NULL" +
		" AND (pps.id IS NULL OR NOT pps.active)"
	if err := db.QueryRow(revokedQ).Scan(&revoked); err != nil {
		t.Fatalf("count revoked primary links: %v", err)
	}
	t.Logf("live active primary sgpps edges with a revoked/unreachable eligibility link: %d", revoked)

	preFix := foldBranchBRows(t, db, "")
	postFix := foldBranchBRows(t, db, classEdgeEligibilityLivePredicate)

	if len(preFix) == 0 {
		t.Skip("dd branch (b) emits zero rows on this database — nothing to compare")
	}

	for k := range postFix {
		if !preFix[k] {
			t.Errorf("post-fix fold emits %q which the pre-fix fold did not — this predicate must only suppress attribution", k)
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
			sort.Strings(lost)
			if len(lost) > 5 {
				lost = lost[:5]
			}
			t.Fatalf("no live eligibility is revoked, so the fold predicate must be a no-op, but %d of %d tuples were suppressed (first few: %v)",
				len(preFix)-len(postFix), len(preFix), lost)
		}
		t.Logf("no-op confirmed: %d branch (b) tuples, identical before and after", len(preFix))
		return
	}
	t.Logf("%d revoked-eligibility primary edges exist; suppression is expected: %d → %d tuples", revoked, len(preFix), len(postFix))
}
