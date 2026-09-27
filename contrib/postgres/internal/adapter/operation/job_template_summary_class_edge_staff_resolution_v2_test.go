//go:build postgresql

package operation

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Parity proof for the M5 courses-fold staff-resolution cutover
// (docs/plan/20260724-section-assignment-merged espyna.md §1b/§5, plan.md §4 M5
// row, consumer 2 of 2 — "the courses display fold"): the dd CTE's class-edge
// branch (b) inside jobTemplateSummaryCTEs moved teacher-name resolution from
// the edge's own legacy staff_id (f10) alone to
// COALESCE(pps.staff_id, e.staff_id) — the edge's linked product_plan_staff
// eligibility row (f13) preferred, legacy f10 as a fallback for rows that
// predate the M3 link-up. Companion to
// contrib/postgres/internal/adapter/principalscope/class_edge_staff_resolution_v2_test.go
// (same COALESCE expression, the other of the two M5 consumers named in the
// task).

// openJTSLiveDB opens the TEST_DATABASE_URL-gated live database, mirroring the
// package's sibling integration-test idiom (job_phase_approval_concurrency_test.go).
func openJTSLiveDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping live courses-fold staff-resolution parity test")
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

// TestJobTemplateSummary_ClassEdgeStaffResolutionV2_COALESCESemantics proves
// the resolution RULE (a)/(b)/(c) directly against literal VALUES rows — the
// same COALESCE(pps_staff_id, legacy_staff_id) expression the dd CTE's branch
// (b) now runs. Nothing is read from or written to any application table (see
// the principalscope companion test for the identical proof against the other
// M5 consumer).
func TestJobTemplateSummary_ClassEdgeStaffResolutionV2_COALESCESemantics(t *testing.T) {
	db := openJTSLiveDB(t)
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
		"linked":           "staff-A", // (a) f13 set, legacy agrees (dual-write)
		"unlinked":         "staff-B", // (b) f13 NULL — falls back to legacy f10
		"v2_authoritative": "staff-C", // (c) f13 set — wins over a stale legacy value
	}
	for name, wantStaff := range want {
		if got[name] != wantStaff {
			t.Errorf("case %q: COALESCE resolved %q, want %q", name, got[name], wantStaff)
		}
	}
}

// legacyOnlyBuildListJobTemplateSummariesSQL uses the current Courses query
// shape, changing only the class-edge staff resolution from the v2
// COALESCE(pps.staff_id, e.staff_id) to the legacy e.staff_id. In particular,
// the P3 delivery fold, price-schedule lookup, approval rollups, and the
// linked-but-revoked eligibility predicate are identical on both sides.
func legacyOnlyBuildListJobTemplateSummariesSQL(workspaceID string) (stmt string, args []any) {
	stmt, args = buildListJobTemplateSummariesSQL(workspaceID, "", "", 0, 0, nil)
	const v2Join = "ON st.id = COALESCE(pps.staff_id, e.staff_id)"
	if strings.Count(stmt, v2Join) != 1 {
		panic("Courses SQL no longer has exactly one v2 class-edge staff join")
	}
	return strings.Replace(stmt, v2Join, "ON st.id = e.staff_id", 1), args
}

// sampleJTSWorkspace picks a real workspace.id that has at least one active
// 'primary' class-edge (sgpps) row — dynamically, never hardcoded — so the
// comparison exercises the branch under test.
func sampleJTSWorkspace(t *testing.T, db *sql.DB) string {
	t.Helper()
	var ws string
	q := "SELECT workspace_id FROM " + entityid.SubscriptionGroupProductPlanStaff +
		" WHERE active AND role = 'primary' GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1"
	if err := db.QueryRow(q).Scan(&ws); err != nil {
		if err == sql.ErrNoRows {
			t.Skip("no active primary class-edge sgpps rows on this database")
		}
		t.Fatalf("sample workspace: %v", err)
	}
	return ws
}

// The post-P3 fold exposes staff arrays rather than one row per staff. Compare
// both arrays and the job count at the public Courses row grain.
func scanSummaryKeyRows(t *testing.T, db *sql.DB, stmt string, args []any) []string {
	t.Helper()
	wrapped := "SELECT job_template_id, subscription_group_id, staff_ids::text, staff_names::text, job_count FROM (" + stmt + ") x WHERE job_template_id IS NOT NULL"
	rows, err := db.Query(wrapped, args...)
	if err != nil {
		t.Fatalf("query: %v\nSQL:\n%s", err, wrapped)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var templateID, groupID, staffID, staffName string
		var jobCount int
		if err := rows.Scan(&templateID, &groupID, &staffID, &staffName, &jobCount); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%d", templateID, groupID, staffID, staffName, jobCount))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	sort.Strings(out)
	return out
}

// TestJobTemplateSummary_ClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly
// is the comparative query test: it runs the OLD (legacy-staff_id-only,
// pre-cutover) FULL summary statement and the NEW (COALESCE, current
// production) FULL summary statement — via the real buildListJobTemplateSummariesSQL
// — for a real, dynamically-sampled workspace on a live database — read-only
// (SELECT only, no INSERT/UPDATE/DELETE) — and asserts the (template, group,
// staff_ids, staff_names, job_count) row sets are byte-identical.
//
// GROUND-TRUTH CORRECTION (2026-07-25, post-M3): this comment previously said
// "every live education1 sgpps row is unlinked (f13 NULL)". M3 has landed and
// inverted that — 379/379 active sgpps rows are f13-linked — but the backfill set
// pps.staff_id equal to e.staff_id in all of them, so COALESCE still returns the
// same value down either branch. This test proves the cutover changed nothing
// observable; it is NOT a proof that the f13 join is correct (audit gap M5-G1).
func TestJobTemplateSummary_ClassEdgeStaffResolutionV2_LiveParityAgainstLegacyOnly(t *testing.T) {
	db := openJTSLiveDB(t)
	defer db.Close()

	workspaceID := sampleJTSWorkspace(t, db)
	t.Logf("sampled workspace=%s", workspaceID)

	oldStmt, oldArgs := legacyOnlyBuildListJobTemplateSummariesSQL(workspaceID)
	newStmt, newArgs := buildListJobTemplateSummariesSQL(workspaceID, "", "", 0, 0, nil)

	oldRows := scanSummaryKeyRows(t, db, oldStmt, oldArgs)
	newRows := scanSummaryKeyRows(t, db, newStmt, newArgs)

	if len(oldRows) == 0 {
		t.Fatal("legacy-only statement returned zero rows — sample is not discriminating, pick a different workspace")
	}
	if len(oldRows) != len(newRows) {
		t.Fatalf("row count differs — legacy-only=%d, v2-cutover=%d\nlegacy=%v\nv2=%v", len(oldRows), len(newRows), oldRows, newRows)
	}
	for i := range oldRows {
		if oldRows[i] != newRows[i] {
			t.Fatalf("rows diverge at index %d — legacy-only=%q, v2-cutover=%q\nlegacy=%v\nv2=%v", i, oldRows[i], newRows[i], oldRows, newRows)
		}
	}
	t.Logf("parity confirmed over %d rows", len(oldRows))
}
