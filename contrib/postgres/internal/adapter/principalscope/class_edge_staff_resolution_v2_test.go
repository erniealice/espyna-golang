//go:build postgresql

package principalscope

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Parity proof for the M5 class-edge sgpps-tier cutover
// (docs/plan/20260724-section-assignment-merged espyna.md §1b/§5, plan.md §4
// M5 row): staff resolution inside reachableJobUnion/reachableClientUnion (and
// their EXISTS/IDs seams) moved from the edge's own legacy staff_id (f10) alone
// to COALESCE(pps.staff_id, e.staff_id) — the edge's linked product_plan_staff
// eligibility row (f13) preferred, legacy f10 as a fallback for rows that
// predate the M3 link-up. This file proves the cutover changed NOTHING
// observable:
//
//   - TestClassEdgeStaffResolutionV2_COALESCESemantics exercises the EXACT
//     COALESCE expression production code runs, against literal VALUES rows (no
//     table is read or written), covering the three cases the task calls for:
//     (a) linked, (b) unlinked-fallback, (c) v2-authoritative-over-stale-legacy.
//   - TestClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly is the
//     comparative query test: it reconstructs the PRE-CUTOVER class-edge branch
//     verbatim and runs it, side-by-side with the actual (current) production
//     function, against REAL live rows — read-only, no writes — and asserts the
//     reachable job/client sets are byte-identical.
//
// GROUND-TRUTH CORRECTION (2026-07-25, post-M3): this file originally recorded
// "every education1 sgpps row is unlinked (f13 NULL, 379/379)". The M3 backfill
// has since landed and INVERTED that: 379/379 active sgpps rows now carry BOTH
// f12 and f13, so the parity test below traverses the LINKED branch, not the
// fallback. It still passes because the backfill set pps.staff_id equal to
// e.staff_id in every row (0 divergent), which is exactly why it has no power to
// discriminate a correct join from a broken one — see audit gap M5-G1. Treat its
// green as "the cutover did not change live answers", never as "the f13 join is
// proven correct".
//
// Both tests are gated on TEST_DATABASE_URL (this package's sibling idiom —
// contrib/postgres/internal/adapter/operation/job_phase_approval_concurrency_test.go)
// and skip cleanly when no live database is reachable.

// openClassEdgeV2LiveDB opens the TEST_DATABASE_URL-gated live database,
// skipping (not failing) when it is unset or unreachable.
func openClassEdgeV2LiveDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping live class-edge staff-resolution parity test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}
	return db
}

// TestClassEdgeStaffResolutionV2_COALESCESemantics proves the resolution RULE
// (a)/(b)/(c) directly against literal VALUES rows — the same
// COALESCE(pps_staff_id, legacy_staff_id) expression production code runs
// (principalscope.go's reachableJobUnion/reachableClientUnion/StaffReachable*
// class-edge branches; job_template_summary_query.go's dd CTE branch (b)).
// Nothing is read from or written to any application table.
func TestClassEdgeStaffResolutionV2_COALESCESemantics(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()

	const q = `SELECT t.name, COALESCE(t.pps_staff_id, t.legacy_staff_id) AS resolved
FROM (VALUES
  ('linked',           'staff-A'::text, 'staff-A'::text),
  ('unlinked',         NULL::text,      'staff-B'::text),
  ('v2_authoritative', 'staff-C'::text, 'staff-STALE'::text)
) AS t(name, pps_staff_id, legacy_staff_id)`

	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var name, resolved string
		if err := rows.Scan(&name, &resolved); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[name] = resolved
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	want := map[string]string{
		// (a) linked (f13 set; dual-write keeps legacy in step): resolves to the
		// SAME staff via either source — byte-identical to the pre-cutover
		// legacy-only read for every row the M2 dual-write has touched.
		"linked": "staff-A",
		// (b) unlinked (predates the M3 link-up; f13 NULL — the LEFT JOIN finds
		// no pps row): falls back to the edge's own legacy staff_id (f10) — the
		// fallback this task adds, removed at M7.
		"unlinked": "staff-B",
		// (c) v2-authoritative (f13 set, legacy stale/disagreeing): the v2 link
		// wins over a merely-present but wrong legacy value — proves f13 is
		// PREFERRED (not just a null-only rescue), so a future M7 legacy-column
		// repurposing cannot silently break resolution.
		"v2_authoritative": "staff-C",
	}
	for name, wantStaff := range want {
		if got[name] != wantStaff {
			t.Errorf("case %q: COALESCE resolved %q, want %q", name, got[name], wantStaff)
		}
	}
	if len(got) != len(want) {
		t.Errorf("want %d cases, scanned %d: %v", len(want), len(got), got)
	}
}

// classEdgeStaffSample is a real (staff.id, workspace.id) pair with at least
// one active sgpps edge reaching at least one job via the class-edge tier —
// picked dynamically off live data (never hardcoded ids), the same idiom as
// job_phase_approval_concurrency_test.go's loadTwoSheets.
type classEdgeStaffSample struct {
	staffID, workspaceID string
	reachableJobs        int
}

// sampleActiveClassEdgeStaff finds the staff/workspace pair with the LARGEST
// class-edge-reachable job set (maximizes the comparison's discriminating
// power) using the exact join shape reachableJobUnion's 4th branch walks,
// independent of the staff-resolution column under test.
func sampleActiveClassEdgeStaff(t *testing.T, db *sql.DB) classEdgeStaffSample {
	t.Helper()
	q := `SELECT e.staff_id, e.workspace_id, COUNT(DISTINCT jce.id) AS n
FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e
JOIN ` + entityid.SubscriptionGroupMember + ` m ON m.subscription_group_id = e.subscription_group_id AND m.active
JOIN ` + entityid.ProductPlan + ` pp ON pp.id = e.product_plan_id
JOIN ` + entityid.Job + ` jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id
WHERE e.active
GROUP BY e.staff_id, e.workspace_id
ORDER BY n DESC
LIMIT 1`
	var s classEdgeStaffSample
	if err := db.QueryRow(q).Scan(&s.staffID, &s.workspaceID, &s.reachableJobs); err != nil {
		if err == sql.ErrNoRows {
			t.Skip("no active class-edge sgpps rows with a reachable job on this database")
		}
		t.Fatalf("sample class-edge staff: %v", err)
	}
	if s.reachableJobs == 0 {
		t.Skip("sampled class-edge staff reaches zero jobs — nothing to compare")
	}
	return s
}

// queryIDSet runs stmt(staffID, wsID) and returns the sorted set of scanned ids.
func queryIDSet(t *testing.T, db *sql.DB, stmt, staffID, wsID string) []string {
	t.Helper()
	rows, err := db.Query(stmt, staffID, wsID)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, stmt)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	sort.Strings(out)
	return out
}

// legacyOnlyReachableJobUnion reconstructs reachableJobUnion EXACTLY as it read
// before this task's cutover (all four tiers; ONLY the 4th, class-edge tier's
// staff predicate differs from the current production function — "e.staff_id =
// s" here vs "COALESCE(pps.staff_id, e.staff_id) = s" in the real function).
// The first three tiers are copied verbatim (byte-for-byte) so the comparison
// in TestClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly isolates
// exactly the one fragment this task changed — comparing a FULL union against
// a FULL union, the same shape every consumer (StaffReachableJobIDsSQL /
// StaffReachableJobClause) actually executes.
func legacyOnlyReachableJobUnion(staffP, wsP int) string {
	s := fmt.Sprintf("$%d", staffP)
	w := fmt.Sprintf("$%d", wsP)
	return "SELECT jp.job_id FROM " + entityid.JobPhase + " jp" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" JOIN " + entityid.Job + " jw ON jw.id = jp.job_id AND jw.workspace_id = " + w +
		" WHERE jt.assigned_to = " + s +
		" UNION " +
		"SELECT jp2.job_id FROM " + entityid.JobPhase + " jp2" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" JOIN " + entityid.Job + " jw2 ON jw2.id = jp2.job_id AND jw2.workspace_id = " + w +
		" WHERE t.recorded_by = " + s + " OR t.reviewed_by = " + s +
		" UNION " +
		"SELECT jw3.id FROM " + entityid.Job + " jw3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = jw3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = jw3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = " + s + " AND ss.status = 'active' AND ss.active = true" +
		" AND jw3.origin_type = '" + originTypeSubscription + "'" +
		" AND jw3.workspace_id = " + w + " AND ss.workspace_id = " + w +
		" UNION " +
		// PRE-CUTOVER class-edge tier (legacy staff_id ONLY — the code this task
		// replaced; the current production tier is at reachableJobUnion below).
		"SELECT jce.id FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = " + s + " AND e.active AND e.workspace_id = " + w
}

// legacyOnlyReachableClientUnion is legacyOnlyReachableJobUnion for
// reachableClientUnion's pre-cutover shape (same four tiers, client_id grain).
func legacyOnlyReachableClientUnion(staffP, wsP int) string {
	s := fmt.Sprintf("$%d", staffP)
	w := fmt.Sprintf("$%d", wsP)
	return "SELECT j.client_id FROM " + entityid.Job + " j" +
		" JOIN " + entityid.JobPhase + " jp ON jp.job_id = j.id" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" WHERE jt.assigned_to = " + s + " AND j.workspace_id = " + w +
		" UNION " +
		"SELECT j2.client_id FROM " + entityid.Job + " j2" +
		" JOIN " + entityid.JobPhase + " jp2 ON jp2.job_id = j2.id" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" WHERE (t.recorded_by = " + s + " OR t.reviewed_by = " + s + ") AND j2.workspace_id = " + w +
		" UNION " +
		"SELECT j3.client_id FROM " + entityid.Job + " j3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = j3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = j3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = " + s + " AND ss.status = 'active' AND ss.active = true" +
		" AND j3.origin_type = '" + originTypeSubscription + "'" +
		" AND j3.workspace_id = " + w + " AND ss.workspace_id = " + w +
		" UNION " +
		// PRE-CUTOVER class-edge tier (legacy staff_id ONLY).
		"SELECT jce.client_id FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = " + s + " AND e.active AND e.workspace_id = " + w
}

// TestClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly is the comparative
// query test: it runs the OLD (legacy-staff_id-only, pre-cutover) FULL
// reachable union and the NEW (COALESCE, current production) FULL reachable
// union — the exact shape every real consumer executes — against a real,
// dynamically-sampled staff/workspace pair on a live database — read-only
// (SELECT only, no INSERT/UPDATE/DELETE) — and asserts the resulting job.id /
// client.id sets are byte-identical sorted lists.
//
// POST-M3 (2026-07-25): 379/379 active sgpps rows are now f13-linked, and the
// backfill made pps.staff_id identical to e.staff_id in every one of them — so
// COALESCE returns the same value down either branch and this test's green is
// non-discriminating by construction (audit gap M5-G1). It proves the cutover
// changed nothing observable; it does NOT prove the join column is right.
// TestClassEdgeStaffResolutionV2_COALESCESemantics covers the divergent cases
// live data cannot express.
func TestClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()

	sample := sampleActiveClassEdgeStaff(t, db)
	t.Logf("sampled staff=%s workspace=%s class-edge-reachable jobs=%d",
		sample.staffID, sample.workspaceID, sample.reachableJobs)

	t.Run("job_ids", func(t *testing.T) {
		oldIDs := queryIDSet(t, db, legacyOnlyReachableJobUnion(1, 2), sample.staffID, sample.workspaceID)
		newIDs := queryIDSet(t, db, reachableJobUnion(1, 2), sample.staffID, sample.workspaceID)
		assertIDSetsEqual(t, "reachableJobUnion", oldIDs, newIDs)
	})

	t.Run("client_ids", func(t *testing.T) {
		oldIDs := queryIDSet(t, db, legacyOnlyReachableClientUnion(1, 2), sample.staffID, sample.workspaceID)
		newIDs := queryIDSet(t, db, reachableClientUnion(1, 2), sample.staffID, sample.workspaceID)
		assertIDSetsEqual(t, "reachableClientUnion", oldIDs, newIDs)
	})
}

// assertIDSetsEqual fails with a readable diff when two sorted id slices are
// not identical.
func assertIDSetsEqual(t *testing.T, label string, legacyOnly, v2Cutover []string) {
	t.Helper()
	if len(legacyOnly) == 0 {
		t.Fatalf("%s: legacy-only query returned zero rows — sample is not discriminating, pick a different staff", label)
	}
	if len(legacyOnly) != len(v2Cutover) {
		t.Fatalf("%s: set size differs — legacy-only=%d, v2-cutover=%d\nlegacy=%v\nv2=%v", label, len(legacyOnly), len(v2Cutover), legacyOnly, v2Cutover)
	}
	for i := range legacyOnly {
		if legacyOnly[i] != v2Cutover[i] {
			t.Fatalf("%s: sets diverge at index %d — legacy-only=%q, v2-cutover=%q\nlegacy=%v\nv2=%v", label, i, legacyOnly[i], v2Cutover[i], legacyOnly, v2Cutover)
		}
	}
}
