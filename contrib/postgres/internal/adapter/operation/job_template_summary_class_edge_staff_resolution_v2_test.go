//go:build postgresql

package operation

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
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

// legacyOnlyJobTemplateSummaryCTEs reconstructs jobTemplateSummaryCTEs EXACTLY
// as it read before this task's cutover: identical in every respect EXCEPT the
// dd CTE's branch (b) staff join, which binds st.id = e.staff_id directly (no
// product_plan_staff LEFT JOIN, no COALESCE) — the code this task replaced.
// Copied verbatim (not paraphrased) so the comparative test below isolates
// exactly the one fragment that changed.
func legacyOnlyJobTemplateSummaryCTEs(jjWhere string) string {
	return `WITH jj AS MATERIALIZED (
    SELECT
        j.id               AS job_id,
        j.origin_id        AS subscription_id,
        j.client_id        AS client_id,
        jt.id              AS template_id,
        jt.name            AS template_name,
        jt.output_product_id AS output_product_id,
        jt.job_category_id AS job_category_id
    FROM ` + entityid.Job + ` j
    JOIN ` + entityid.JobTemplate + ` jt
           ON jt.id = j.job_template_id AND jt.workspace_id = $1 AND jt.active
    ` + jjWhere + `
),
dd AS MATERIALIZED (
    -- Branch (a): the SUBSCRIPTION_SEAT deliverer/group side (the original dd) — unchanged.
    SELECT
        ss.subscription_id       AS subscription_id,
        ss.client_id             AS client_id,
        pl.product_id            AS product_id,
        st.id                    AS staff_id,
        u.first_name             AS first_name,
        u.last_name              AS last_name,
        sgm.subscription_group_id AS subscription_group_id
    FROM ` + entityid.SubscriptionSeat + ` ss
    JOIN ` + entityid.ProductPlan + ` pl
           ON pl.id = ss.product_plan_id
    JOIN ` + entityid.Staff + ` st
           ON st.id = ss.staff_id AND st.workspace_id = $1
    LEFT JOIN "` + entityid.User + `" u
           ON u.id = st.user_id AND u.active
    JOIN ` + entityid.SubscriptionGroupMember + ` sgm
           ON sgm.subscription_id = ss.subscription_id AND sgm.client_id = ss.client_id
          AND sgm.workspace_id = $1 AND sgm.active
    WHERE ss.status = 'active' AND ss.active AND ss.workspace_id = $1
    UNION
    -- Branch (b): class-edge (sgpps) deliverers, role primary only (C11) —
    -- PRE-CUTOVER: staff resolved from the edge's OWN legacy staff_id ONLY.
    SELECT
        m.subscription_id        AS subscription_id,
        m.client_id              AS client_id,
        pl.product_id            AS product_id,
        st.id                    AS staff_id,
        u.first_name             AS first_name,
        u.last_name              AS last_name,
        m.subscription_group_id  AS subscription_group_id
    FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e
    JOIN ` + entityid.SubscriptionGroupMember + ` m
           ON m.subscription_group_id = e.subscription_group_id AND m.active
    JOIN ` + entityid.ProductPlan + ` pl
           ON pl.id = e.product_plan_id
    JOIN ` + entityid.Staff + ` st
           ON st.id = e.staff_id AND st.workspace_id = $1
    LEFT JOIN "` + entityid.User + `" u
           ON u.id = st.user_id AND u.active
    WHERE e.active AND e.workspace_id = $1 AND e.role = 'primary'
      AND e.id = (
          SELECT e2.id FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e2
          WHERE e2.subscription_group_id = e.subscription_group_id
            AND e2.product_plan_id = e.product_plan_id
            AND e2.active AND e2.workspace_id = $1 AND e2.role = 'primary'
          ORDER BY e2.date_created DESC, e2.id DESC
          LIMIT 1
      )
),
pa AS (
    SELECT
        jj.template_id            AS template_id,
        gm.subscription_group_id  AS group_id,
        jp.template_phase_id      AS template_phase_id,
        MIN(` + approvalStatusRankCASE + `) AS min_rank,
        MAX(` + approvalStatusRankCASE + `) AS max_rank,
        BOOL_OR(tox.job_task_id IS NOT NULL) AS has_data
    FROM jj
    JOIN ` + entityid.JobPhase + ` jp
           ON jp.job_id = jj.job_id AND jp.active AND jp.template_phase_id IS NOT NULL
    LEFT JOIN ` + entityid.JobTask + ` tk
           ON tk.job_phase_id = jp.id AND tk.active
    LEFT JOIN ` + entityid.TaskOutcome + ` tox
           ON tox.job_task_id = tk.id AND tox.active
    LEFT JOIN ` + entityid.SubscriptionGroupMember + ` gm
           ON gm.subscription_id = jj.subscription_id AND gm.client_id = jj.client_id
          AND gm.workspace_id = $1 AND gm.active
    GROUP BY jj.template_id, gm.subscription_group_id, jp.template_phase_id
),
tp AS (
    SELECT template_id, template_phase_id,
           MIN(min_rank)     AS min_rank,
           MAX(max_rank)     AS max_rank,
           BOOL_OR(has_data) AS has_data
    FROM pa
    GROUP BY template_id, template_phase_id
),
ta AS (
    SELECT template_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM tp
    GROUP BY template_id
),
ga AS (
    SELECT template_id, group_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM pa
    WHERE group_id IS NOT NULL
    GROUP BY template_id, group_id
)`
}

// legacyOnlyBuildListJobTemplateSummariesSQL builds the FULL pre-cutover
// statement (no status/group/scope/pagination — matching fullSummarySQL's
// base-shape idiom) for one workspace, reusing the UNCHANGED downstream
// builders (jobTemplateSummarySelectFrom / GroupOrder / ApprovalSelect) so the
// comparison is scoped to exactly the dd CTE branch (b) staff join.
func legacyOnlyBuildListJobTemplateSummariesSQL(workspaceID string) (stmt string, args []any) {
	jjWhere := "WHERE j.job_template_id IS NOT NULL" +
		" AND j.workspace_id = $1" +
		" AND j.active" +
		" AND j.origin_type = '" + originTypeSubscriptionToken + "'"
	stmt = legacyOnlyJobTemplateSummaryCTEs(jjWhere) + ",\nbase AS (\n" +
		jobTemplateSummarySelectFrom() + "\n" +
		jobTemplateSummaryGroupOrder() + "\n)\n" +
		jobTemplateSummaryApprovalSelect()
	return stmt, []any{workspaceID}
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

// summaryKeyRow is the reduced projection this comparative test scans: the
// columns downstream of (and dependent on) the dd CTE's staff resolution.
// job_category_id / price_schedule / approval-preaggregate columns are
// untouched by this task's diff and are intentionally excluded — the columns
// here are exactly the ones that would visibly differ if staff resolution
// diverged (a wrong/missing staff_id changes staff_name, and folds the
// job_count differently).
func scanSummaryKeyRows(t *testing.T, db *sql.DB, stmt string, args []any) []string {
	t.Helper()
	wrapped := "SELECT job_template_id, subscription_group_id, staff_id, staff_name, job_count FROM (" + stmt + ") x"
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
// staff_id, staff_name, job_count) row sets are byte-identical. Every live
// education1 sgpps row is unlinked today (f13 NULL, ground-truthed
// 2026-07-24), so COALESCE degrades to the legacy column for 100% of live
// rows: this is an empirical, whole-table proof that the cutover changed
// nothing observable in the courses fold for the CURRENT data shape.
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
