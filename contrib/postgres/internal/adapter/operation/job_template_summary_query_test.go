//go:build postgresql

package operation

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// fullSummarySQL is the whole two-CTE statement (jj + dd CTEs + outer SELECT),
// the unit the structural assertions read now that the joins live across
// jobTemplateSummaryCTEs() and jobTemplateSummarySelectFrom(). Built with no
// status/group/scope/pagination so the base shape is exercised.
func fullSummarySQL() string {
	stmt, _ := buildListJobTemplateSummariesSQL("ws-1", "", "", 0, 0, nil)
	return stmt
}

// TestJobTemplateSummarySQL_TableNamesFromEntityID locks the
// infra-sql-table-name-source rule: EVERY table identifier in the aggregate
// comes from a registry/entityid constant, never a hand-typed literal. If a
// constant's value ever changes, this test still passes (it reads the same
// constants), but a stray quoted literal or a wrong table (e.g. the
// subscription_group_product_plan_staff over-count trap the S2 note warns
// against) is caught. Post-20260718-perf: the joins are split across the jj/dd
// MATERIALIZED CTEs and the outer SELECT, so the whole statement is asserted.
func TestJobTemplateSummarySQL_TableNamesFromEntityID(t *testing.T) {
	sql := fullSummarySQL()

	// Every joined table must be present, sourced from its entityid constant.
	mustJoin := []struct {
		alias string
		table string
	}{
		{"j", entityid.Job},                       // jj CTE
		{"jt", entityid.JobTemplate},              // jj CTE
		{"sgm", entityid.SubscriptionGroupMember}, // dd CTE, seat branch
		{"sg", entityid.SubscriptionGroup},        // outer
		{"ps", entityid.PriceSchedule},            // outer
		{"ss", entityid.SubscriptionSeat},         // dd CTE, seat branch
		{"pl", entityid.ProductPlan},              // dd CTE
		{"st", entityid.Staff},                    // dd CTE
		{"op", entityid.Product},                  // outer
		{"jp", entityid.JobPhase},                 // pa CTE (R7 P4)
		{"tk", entityid.JobTask},                  // pa CTE (R7 P4)
		{"tox", entityid.TaskOutcome},             // pa CTE (R7 P4)
		{"gm", entityid.SubscriptionGroupMember},  // pa CTE (R7 P4)
		// dd CTE, class-edge (sgpps) branch (C11): the class edge itself + a
		// SECOND subscription_group_member alias (m) that carries the (subscription,
		// client, group) triple for a section that has servicers but zero seats.
		{"e", entityid.SubscriptionGroupProductPlanStaff}, // class_primary_edges CTE
		{"m", entityid.SubscriptionGroupMember},           // dd CTE, class-edge branch
		// v2 cutover: the class-edge branch's staff resolution LEFT JOINs the
		// eligibility table (product_plan_staff, f13) — docs/plan/20260724-
		// section-assignment-merged espyna.md §1b/M5.
		{"pps", entityid.ProductPlanStaff}, // dd CTE, class-edge branch, v2 eligibility link
	}
	for _, m := range mustJoin {
		if !strings.Contains(sql, " "+m.table+" "+m.alias) {
			t.Errorf("statement missing %q aliased %q\nSQL:\n%s", m.table, m.alias, sql)
		}
	}

	// Both CTEs must be MATERIALIZED (the structural hash-join pin — P1 win).
	for _, cte := range []string{"jj AS MATERIALIZED", "dd AS MATERIALIZED"} {
		if !strings.Contains(sql, cte) {
			t.Errorf("missing %q — the MATERIALIZED plan pin is load-bearing\nSQL:\n%s", cte, sql)
		}
	}

	// The jj×dd hash join must key on all three columns (restores the original
	// seat/plan-product ↔ template-output inner-join semantics).
	for _, key := range []string{
		"dd.subscription_id = jj.subscription_id",
		"dd.client_id = jj.client_id",
		"dd.product_id = jj.output_product_id",
	} {
		if !strings.Contains(sql, key) {
			t.Errorf("jj×dd join missing key %q\nSQL:\n%s", key, sql)
		}
	}

	// "user" is the one reserved-word identifier — must be double-quoted, from
	// the entityid.User constant (matches the staff CTE / outcome_matrix join).
	if !strings.Contains(sql, `"`+entityid.User+`" u`) {
		t.Errorf("SELECT/FROM missing double-quoted %q join\nSQL:\n%s", entityid.User, sql)
	}

	// C11: the class-edge (sgpps) branch is now REQUIRED in the dd CTE (a section
	// with servicers but no seats otherwise yields zero /courses + /report-cards
	// rows). It is grain-correct — NOT the old cohort-grain over-count trap —
	// BECAUSE it joins through subscription_group_member to reach the actual
	// (subscription, client) memberships, exactly the seat branch's grain. The
	// specific join chain + the role='primary' record semantic are locked in
	// TestJobTemplateSummarySQL_ClassEdgeDelivererBranch below.
	if !strings.Contains(sql, entityid.SubscriptionGroupProductPlanStaff+" e") {
		t.Errorf("dd CTE missing the class-edge branch (sgpps aliased e) — C11 zero-rows regression\nSQL:\n%s", sql)
	}
}

// TestJobTemplateSummarySQL_ClassEdgeDelivererBranch locks the C11 fix: the dd
// CTE is a UNION of (a) the subscription_seat branch and (b) a class-edge (sgpps)
// branch, so a section that has class-edge servicers but ZERO seats (the AY-2627
// shape) still produces a dd row and its jobs survive the jj⋈dd INNER join. The
// branch:
//   - joins sgpps e → subscription_group_member m (on the group, active) →
//     product_plan pl (the edge's subject plan) → product_plan_staff pps (LEFT,
//     the v2 eligibility link, f13) → staff st (workspace-bound, resolved via
//     COALESCE(pps.staff_id, e.staff_id) — v2-linked preferred, legacy f10
//     fallback for rows predating the M3 link-up, docs/plan/20260724-section-
//     assignment-merged espyna.md §1b/M5) → "user" u (LEFT, active), sourcing
//     table names from entityid constants;
//   - filters role='primary' ONLY — the teacher-of-record RECORD semantic (§B):
//     a 'primary' edge generates the class row + the Teacher/Deliverer column; an
//     'access' edge is visibility-only (principalscope) and must NOT surface here;
//   - picks newest primary edges set-wise in class_primary_edges, then binds
//     the staff ($1);
//   - projects the member-sourced (subscription_id, client_id, subscription_group_id)
//   - edge-plan product_id so the emitted column set is byte-identical to the
//     seat branch (jj⋈dd + collateDeliverySummaries need no change);
//   - dedupes via UNION (not UNION ALL): a (sub, client, product, staff) pair
//     reachable through BOTH a seat and a class edge collapses to one row.
func TestJobTemplateSummarySQL_ClassEdgeDelivererBranch(t *testing.T) {
	sql := fullSummarySQL()

	// The class-edge join chain, each ON clause sourced off the picked edge (e).
	for _, frag := range []string{
		entityid.SubscriptionGroupProductPlanStaff + " e",
		entityid.SubscriptionGroupMember + " m",
		"m.subscription_group_id = e.subscription_group_id AND m.active",
		"pl.id = e.product_plan_id",
		"LEFT JOIN " + entityid.ProductPlanStaff + " pps", // v2 eligibility link is OPTIONAL (f13 may be unset pre-M3)
		"pps.id = e.product_plan_staff_id",
		"st.id = COALESCE(pps.staff_id, e.staff_id) AND st.workspace_id = $1",
		`"` + entityid.User + `" u`,
		"u.id = st.user_id AND u.active",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("class-edge branch missing join fragment %q\nSQL:\n%s", frag, sql)
		}
	}

	// role='primary' RECORD filter + workspace bind are applied while selecting
	// primary edges. The visibility-only 'access' role must NEVER surface here.
	if !strings.Contains(sql, "class_primary_edges AS MATERIALIZED") ||
		!strings.Contains(sql, "WHERE e.active AND e.workspace_id = $1 AND e.role = 'primary'") {
		t.Errorf("class-primary CTE missing active/workspace/role='primary' selection\nSQL:\n%s", sql)
	}
	if strings.Contains(sql, "role = 'access'") || strings.Contains(sql, "role='access'") {
		t.Errorf("class-edge branch must NOT filter on 'access' edges — those are visibility-only\nSQL:\n%s", sql)
	}

	// M5-G5: the eligibility-liveness gate must ride the dd WHERE after the
	// class-primary pick, so a
	// linked-but-REVOKED product_plan_staff row stops attributing its staff
	// instead of falling through to the still-dual-written legacy f10 column.
	// Shape assertion only — the behavioural truth table, the naive-join-fix
	// counter-example and the live no-op proof are in
	// job_template_summary_class_edge_eligibility_live_test.go.
	if !strings.Contains(sql, classEdgeEligibilityLivePredicate) {
		t.Errorf("class-edge branch missing the eligibility-liveness gate %q\nSQL:\n%s", classEdgeEligibilityLivePredicate, sql)
	}
	// It must NOT be pushed into the LEFT JOIN condition: there it would null the
	// pps row out and let COALESCE re-grant the revoked staff via legacy f10.
	if strings.Contains(sql, "pps.id = e.product_plan_staff_id AND pps.active") {
		t.Errorf("eligibility-liveness moved into the LEFT JOIN condition — that form is a no-op against the legacy f10 fallback; keep it in the WHERE\nSQL:\n%s", sql)
	}

	// CF-3: two DIFFERENT active primary edges for one (group, product_plan) must
	// collapse to a SINGLE deterministic pick (newest date_created, id breaks ties)
	// so the deliverer is stable across renders and agrees with the grade-sheet's
	// class-edge teacher. DISTINCT ON does this set-wise; do not restore a
	// per-row correlated pick.
	for _, frag := range []string{
		"class_primary_edges AS MATERIALIZED",
		"SELECT DISTINCT ON (e.subscription_group_id, e.product_plan_id)",
		"ORDER BY e.subscription_group_id, e.product_plan_id,",
		"e.date_created DESC, e.id DESC",
		"FROM class_primary_edges e",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("class-edge branch missing CF-3 deterministic-pick fragment %q\nSQL:\n%s", frag, sql)
		}
	}
	for _, forbidden := range []string{
		"e.id = (\n          SELECT e2.id",
		"e2.subscription_group_id = e.subscription_group_id",
	} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("class-primary selection must be set-oriented, found correlated pick %q\nSQL:\n%s", forbidden, sql)
		}
	}
	// The CTE selects the newest primary edge without eligibility filtering. The
	// gate remains in dd, after FROM class_primary_edges e, so a newest revoked
	// edge suppresses rather than falls back to an older primary.
	if !strings.Contains(sql, "FROM class_primary_edges e") ||
		strings.Contains(sql, "WHERE e.active AND e.workspace_id = $1 AND e.role = 'primary'\n      "+classEdgeEligibilityLivePredicate) {
		t.Errorf("eligibility gate must be applied after, not within, class-primary selection\nSQL:\n%s", sql)
	}

	// The branch projects the member-sourced keys (m.*) + the edge-plan product so
	// the emitted 7-column shape matches the seat branch exactly. m.subscription_id
	// and m.client_id appear ONLY in this branch's SELECT (the seat branch reads
	// ss.*/sgm.*), so their presence confirms the member-sourced projection.
	for _, col := range []string{"m.subscription_id", "m.client_id", "m.subscription_group_id"} {
		if !strings.Contains(sql, col) {
			t.Errorf("class-edge branch missing member-sourced projection %q\nSQL:\n%s", col, sql)
		}
	}

	// dd dedupes via UNION (set semantics), never UNION ALL — a (sub, client,
	// product, staff) pair reachable through both a seat and a class edge collapses.
	if !strings.Contains(sql, "UNION") {
		t.Errorf("dd CTE must UNION the seat + class-edge branches\nSQL:\n%s", sql)
	}
	if strings.Contains(sql, "UNION ALL") {
		t.Errorf("dd must UNION (not UNION ALL) so seat/class-edge duplicates dedupe\nSQL:\n%s", sql)
	}

	// The class-edge branch must NOT reintroduce the cohort-grain over-count: it
	// reaches memberships through subscription_group_member (m), NOT by attributing
	// the whole cohort to the edge. It also must not join subscription_seat inside
	// the class-edge branch (that branch exists precisely for the zero-seat shape).
	if !strings.Contains(sql, "FROM class_primary_edges e\n") {
		t.Errorf("picked class-primary edges must drive the class-edge branch (aliased e)\nSQL:\n%s", sql)
	}

	// Workspace-scoping of the class edge is explicit in class_primary_edges ($1); the member m is scoped
	// transitively (globally-unique group FK + the workspace-bound edge + the outer
	// sg join re-gate), so a bind there is neither required nor present.
	if !strings.Contains(sql, "e.workspace_id = $1") {
		t.Errorf("class-edge branch must bind e.workspace_id = $1\nSQL:\n%s", sql)
	}
}

// TestJobTemplateSummarySQL_WorkspacePredicateOnEveryScopedTable locks the
// multi-tenancy invariant: every joined table that CARRIES a workspace_id
// column is bound to $1. product_plan and "user" carry no workspace_id column
// (verified against the live schema) and MUST NOT be bound — they are scoped
// transitively through the workspace-bound seat/template/staff.
func TestJobTemplateSummarySQL_WorkspacePredicateOnEveryScopedTable(t *testing.T) {
	sql := fullSummarySQL()

	// The base job predicate binds j.workspace_id in the jj CTE WHERE; every
	// other workspace-bearing joined table also carries $1 (across the CTEs and
	// the outer SELECT). gm is the pa CTE's subscription_group_member join (R7
	// P4); job_phase/job_task/task_outcome carry no workspace_id column and are
	// bound transitively through the workspace-scoped jj job set.
	boundAliases := []string{"j", "jt", "sgm", "sg", "ps", "ss", "st", "op", "gm"}
	for _, a := range boundAliases {
		if !strings.Contains(sql, a+".workspace_id = $1") {
			t.Errorf("table alias %q missing workspace_id = $1 bind\nSQL:\n%s", a, sql)
		}
	}

	// product_plan (pl), user (u), job_phase (jp), job_task (tk) and
	// task_outcome (tox) have no workspace_id column — a bind there would be a
	// schema error (the phase/task/outcome tables scope transitively via jj).
	for _, banned := range []string{"pl.workspace_id", "u.workspace_id", "jp.workspace_id", "tk.workspace_id", "tox.workspace_id"} {
		if strings.Contains(sql, banned) {
			t.Errorf("SQL binds %q but that table has no workspace_id column\nSQL:\n%s", banned, sql)
		}
	}
}

// TestJobTemplateSummarySQL_PrincipalScopeSeam locks that the principalscope
// clause is spliced at the CORRECT placeholder index and that a non-staff
// (empty) scope leaves the query unscoped (admin sees all). This is the seam
// that keeps STAFF principals confined to their reachable jobs.
func TestJobTemplateSummarySQL_PrincipalScopeSeam(t *testing.T) {
	const scopeMarker = " AND j.id IN (/*scope*/ SELECT 1)"

	t.Run("scope_starts_after_workspace_and_status", func(t *testing.T) {
		var gotStart int
		scopeFn := func(start int) (string, []any) {
			gotStart = start
			return scopeMarker, []any{"staff-1", "ws-1"}
		}
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "", 0, 0, scopeFn)

		if gotStart != 3 {
			t.Errorf("scope start param: want 3 ($1=ws,$2=status), got %d", gotStart)
		}
		if !strings.Contains(stmt, scopeMarker) {
			t.Errorf("scope clause not spliced into stmt:\n%s", stmt)
		}
		wantArgs := []any{"ws-1", "JOB_STATUS_ACTIVE", "staff-1", "ws-1"}
		assertArgs(t, args, wantArgs)
	})

	t.Run("scope_starts_after_group_filter", func(t *testing.T) {
		var gotStart int
		scopeFn := func(start int) (string, []any) {
			gotStart = start
			return scopeMarker, []any{"staff-1", "ws-1"}
		}
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "grp-1", 0, 0, scopeFn)

		if gotStart != 4 {
			t.Errorf("scope start param with group: want 4 ($1=ws,$2=status,$3=group), got %d", gotStart)
		}
		if !strings.Contains(stmt, "sg.id = $3") {
			t.Errorf("group filter not at $3:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1", "JOB_STATUS_ACTIVE", "grp-1", "staff-1", "ws-1"})
	})

	t.Run("empty_scope_leaves_query_unscoped", func(t *testing.T) {
		// non-staff principal: scopeFn returns "" — the aggregate must not gain a
		// row-narrowing IN clause (admin/registrar sees all).
		scopeFn := func(start int) (string, []any) { return "", nil }
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "", 0, 0, scopeFn)

		if strings.Contains(stmt, "j.id IN") {
			t.Errorf("empty scope must not add a j.id IN narrowing clause:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1", "JOB_STATUS_ACTIVE"})
	})

	t.Run("nil_scope_fn_is_safe", func(t *testing.T) {
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "", "", 0, 0, nil)
		if strings.Contains(stmt, "j.id IN") {
			t.Errorf("nil scopeFn must not add a narrowing clause:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1"})
	})
}

// TestJobTemplateSummarySQL_FiltersAndPagination locks the optional status /
// group filters and the pagination placeholders + arg ordering.
func TestJobTemplateSummarySQL_FiltersAndPagination(t *testing.T) {
	t.Run("status_bound_at_2", func(t *testing.T) {
		stmt, _ := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "", 0, 0, nil)
		if !strings.Contains(stmt, "j.status = $2") {
			t.Errorf("status not bound at $2:\n%s", stmt)
		}
	})

	t.Run("empty_status_omits_predicate", func(t *testing.T) {
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "", "", 0, 0, nil)
		if strings.Contains(stmt, "j.status") {
			t.Errorf("empty status must omit the status predicate:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1"})
	})

	t.Run("pagination_placeholders_and_args", func(t *testing.T) {
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "", 50, 100, nil)
		// ws=$1, status=$2, then LIMIT $3 OFFSET $4.
		if !strings.Contains(stmt, "LIMIT $3 OFFSET $4") {
			t.Errorf("pagination placeholders wrong:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1", "JOB_STATUS_ACTIVE", int32(50), int32(100)})
	})

	t.Run("origin_type_and_distinct_count_present", func(t *testing.T) {
		stmt, _ := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "", 0, 0, nil)
		if !strings.Contains(stmt, "j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'") {
			t.Errorf("subscription origin-type predicate missing:\n%s", stmt)
		}
		// COUNT(DISTINCT) is computed at the response row grain before the
		// deliverer arrays are joined, so staff never multiplies a roster count.
		if !strings.Contains(stmt, "COUNT(DISTINCT job_id) AS job_count") {
			t.Errorf("DISTINCT job count missing:\n%s", stmt)
		}
		for _, frag := range []string{
			"delivery AS MATERIALIZED",
			"delivery_staff AS (",
			"ARRAY_AGG(staff_id ORDER BY staff_name, staff_id) AS staff_ids",
			"ARRAY_AGG(staff_name ORDER BY staff_name, staff_id) AS staff_names",
		} {
			if !strings.Contains(stmt, frag) {
				t.Errorf("summary-grain delivery fold missing %q:\n%s", frag, stmt)
			}
		}
		// The LOCKED view order (group name, template name) — since R7 P4 the
		// sort sits ABOVE the base wrapper, keyed through base's output aliases
		// (same key semantics as the pre-P4 `sg.name, jj.template_name`).
		if !strings.Contains(stmt, "ORDER BY selected_rows.subscription_group_name") &&
			!strings.Contains(stmt, "ORDER BY subscription_group_name ASC NULLS FIRST, job_template_name ASC") {
			t.Errorf("locked ORDER BY (group, template) prefix missing:\n%s", stmt)
		}
		if !strings.Contains(stmt, "subscription_group_id ASC NULLS FIRST, job_template_id ASC,") {
			t.Errorf("stable summary-row ORDER BY tie-breakers missing:\n%s", stmt)
		}
	})
}

func TestJobTemplateSummarySQL_ServerPageContract(t *testing.T) {
	t.Run("category_search_sort_and_exact_page_are_bound", func(t *testing.T) {
		stmt, args, err := buildListJobTemplateSummariesRequestSQL(
			"ws-1", "JOB_STATUS_ACTIVE", "", 20, 40,
			jobTemplateSummaryQueryOptions{
				jobCategoryID: "cat-1",
				search: &commonpb.SearchRequest{
					Query:   "50%_\\",
					Options: &commonpb.SearchOptions{SearchFields: []string{"name", "deliverer"}},
				},
				sort: &commonpb.SortRequest{Fields: []*commonpb.SortField{{
					Field: "group", Direction: commonpb.SortDirection_DESC, NullOrder: commonpb.NullOrder_NULLS_LAST,
				}}},
			},
			nil,
		)
		if err != nil {
			t.Fatalf("build request SQL: %v", err)
		}
		for _, fragment := range []string{
			"job_category_id = $3",
			"job_template_name)::text, '') ILIKE $4 ESCAPE '\\'",
			"array_to_string(staff_names, ' '))::text, '') ILIKE $4 ESCAPE '\\'",
			"ORDER BY subscription_group_name DESC NULLS LAST",
			"LIMIT $5 OFFSET $6",
		} {
			if !strings.Contains(stmt, fragment) {
				t.Errorf("request SQL missing %q\nSQL:\n%s", fragment, stmt)
			}
		}
		assertArgs(t, args, []any{"ws-1", "JOB_STATUS_ACTIVE", "cat-1", "%50\\%\\_\\\\%", int32(20), int32(40)})
	})

	t.Run("metadata_precedes_selection_and_page", func(t *testing.T) {
		stmt, _, err := buildListJobTemplateSummariesRequestSQL(
			"ws-1", "JOB_STATUS_ACTIVE", "", 20, 0,
			jobTemplateSummaryQueryOptions{jobCategoryID: "cat-1"}, nil,
		)
		if err != nil {
			t.Fatalf("build request SQL: %v", err)
		}
		countsAt := strings.Index(stmt, "category_counts AS MATERIALIZED")
		selectedAt := strings.Index(stmt, "selected_rows AS MATERIALIZED")
		pageAt := strings.Index(stmt, "page_base AS MATERIALIZED")
		if countsAt < 0 || selectedAt <= countsAt || pageAt <= selectedAt {
			t.Fatalf("category counts must precede selected rows and page\nSQL:\n%s", stmt)
		}
		for _, fragment := range []string{
			"(SELECT COUNT(*) FROM selected_rows) AS total_items",
			"'summary_count', category_counts.summary_count",
			"FROM metadata\nLEFT JOIN page_enriched ON TRUE",
		} {
			if !strings.Contains(stmt, fragment) {
				t.Errorf("metadata SQL missing %q\nSQL:\n%s", fragment, stmt)
			}
		}
	})

	t.Run("fallback_is_explicit_active_only_and_category_wide", func(t *testing.T) {
		active, _, err := buildListJobTemplateSummariesRequestSQL(
			"ws-1", "JOB_STATUS_ACTIVE", "", 20, 0,
			jobTemplateSummaryQueryOptions{includeTemplateFallback: true}, nil,
		)
		if err != nil {
			t.Fatalf("build active fallback SQL: %v", err)
		}
		for _, fragment := range []string{
			"fallback_base AS MATERIALIZED",
			"FROM " + entityid.JobTemplate + " jt",
			"jt.workspace_id = $1 AND jt.active",
			"base.job_category_id IS NOT DISTINCT FROM jt.job_category_id",
			"true                   AS template_grain_fallback",
		} {
			if !strings.Contains(active, fragment) {
				t.Errorf("active fallback SQL missing %q\nSQL:\n%s", fragment, active)
			}
		}

		for name, statusGroup := range map[string][2]string{
			"non_active_status": {"JOB_STATUS_COMPLETED", ""},
			"group_narrowed":    {"JOB_STATUS_ACTIVE", "group-1"},
		} {
			t.Run(name, func(t *testing.T) {
				stmt, _, err := buildListJobTemplateSummariesRequestSQL(
					"ws-1", statusGroup[0], statusGroup[1], 20, 0,
					jobTemplateSummaryQueryOptions{includeTemplateFallback: true}, nil,
				)
				if err != nil {
					t.Fatalf("build SQL: %v", err)
				}
				if strings.Contains(stmt, "fallback_base AS MATERIALIZED") {
					t.Fatalf("%s must not add template fallback\nSQL:\n%s", name, stmt)
				}
			})
		}
	})

	t.Run("unknown_request_fields_fail_closed", func(t *testing.T) {
		_, _, err := buildListJobTemplateSummariesRequestSQL(
			"ws-1", "JOB_STATUS_ACTIVE", "", 20, 0,
			jobTemplateSummaryQueryOptions{sort: &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "name; DROP TABLE job"}}}}, nil,
		)
		if err == nil || !strings.Contains(err.Error(), "unknown summary sort field") {
			t.Fatalf("sort injection error = %v", err)
		}

		_, _, err = buildListJobTemplateSummariesRequestSQL(
			"ws-1", "JOB_STATUS_ACTIVE", "", 20, 0,
			jobTemplateSummaryQueryOptions{search: &commonpb.SearchRequest{
				Query: "x", Options: &commonpb.SearchOptions{SearchFields: []string{"secret_column"}},
			}}, nil,
		)
		if err == nil || !strings.Contains(err.Error(), "unknown summary search field") {
			t.Fatalf("search field error = %v", err)
		}
	})
}

func TestJobTemplateSummaryPaginationBounds(t *testing.T) {
	t.Run("nil_and_zero_limit_remain_unpaginated", func(t *testing.T) {
		for _, p := range []*commonpb.PaginationRequest{nil, {Limit: 0}} {
			limit, offset, err := paginationBounds(p)
			if err != nil || limit != 0 || offset != 0 {
				t.Fatalf("paginationBounds(%+v) = (%d,%d,%v), want (0,0,nil)", p, limit, offset, err)
			}
		}
	})

	t.Run("requested_limit_is_clamped_on_a_copy", func(t *testing.T) {
		p := &commonpb.PaginationRequest{
			Limit: 500,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 2},
			},
		}
		limit, offset, err := paginationBounds(p)
		if err != nil || limit != maxJobTemplateSummaryLimit || offset != maxJobTemplateSummaryLimit {
			t.Fatalf("paginationBounds() = (%d,%d,%v), want (%d,%d,nil)", limit, offset, err, maxJobTemplateSummaryLimit, maxJobTemplateSummaryLimit)
		}
		if p.GetLimit() != 500 {
			t.Fatalf("paginationBounds mutated caller limit to %d", p.GetLimit())
		}
	})

	t.Run("oversized_offset_is_rejected", func(t *testing.T) {
		p := &commonpb.PaginationRequest{
			Limit: 100,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 10_002},
			},
		}
		if _, _, err := paginationBounds(p); err == nil || !strings.Contains(err.Error(), "offset exceeds maximum") {
			t.Fatalf("paginationBounds() error = %v, want oversized-offset rejection", err)
		}
	})

	t.Run("cursor_mode_is_rejected", func(t *testing.T) {
		p := &commonpb.PaginationRequest{
			Limit: 20,
			Method: &commonpb.PaginationRequest_Cursor{
				Cursor: &commonpb.CursorPagination{Token: "offset:0"},
			},
		}
		if _, _, err := paginationBounds(p); err == nil || !strings.Contains(err.Error(), "cursor pagination is not supported") {
			t.Fatalf("paginationBounds() error = %v, want cursor rejection", err)
		}
	})

	t.Run("service_wraps_rejection_before_query", func(t *testing.T) {
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "ws-1"})
		req := &summarypb.ListJobTemplateSummariesRequest{Pagination: &commonpb.PaginationRequest{
			Limit: 20,
			Method: &commonpb.PaginationRequest_Cursor{
				Cursor: &commonpb.CursorPagination{Token: "offset:0"},
			},
		}}
		_, err := (&PostgresJobTemplateSummaryQuery{}).ListJobTemplateSummaries(ctx, req)
		if err == nil || !strings.Contains(err.Error(), "job_template_summary: pagination:") {
			t.Fatalf("ListJobTemplateSummaries() error = %v, want wrapped pagination rejection", err)
		}
	})
}

// TestJobTemplateSummarySQL_JobCategoryProjection locks the R9 W-A1 category
// column-add (plan 20260719-report-cards-landing §3.1): job_template.job_category_id
// (the AUTHORITATIVE category FK, §3.0) is projected inside the jj CTE (where jt is
// ALREADY joined — zero new join/statement), carried through the outer SELECT off
// the jj CTE, and appended to GROUP BY. Because the category is functionally
// determined by the template, it widens the row without fanning it out — the row
// grain and the LOCKED ORDER BY are unchanged.
func TestJobTemplateSummarySQL_JobCategoryProjection(t *testing.T) {
	sql := fullSummarySQL()

	// Projected in the jj CTE off the ALREADY-joined jt (no new join added).
	if !strings.Contains(sql, "jt.job_category_id AS job_category_id") {
		t.Errorf("jj CTE missing jt.job_category_id projection\nSQL:\n%s", sql)
	}
	// It is carried into delivery and the summary-grain counts GROUP BY.
	for _, frag := range []string{"jj.job_category_id             AS job_category_id", "job_category_id, COUNT(DISTINCT job_id) AS job_count"} {
		if !strings.Contains(sql, frag) {
			t.Errorf("category missing summary-grain carry %q\nSQL:\n%s", frag, sql)
		}
	}

	// ZERO new joins: the category column-add must NOT join the job_category table
	// (the category NAME/sort_order are resolved in the view layer, not here). Only
	// the two MATERIALIZED CTEs' existing joins may reference tables.
	if strings.Contains(sql, " "+entityid.JobCategory+" ") {
		t.Errorf("category column-add must NOT join the %q table — name lives in the view layer\nSQL:\n%s",
			entityid.JobCategory, sql)
	}
	// The MATERIALIZED plan pins and the LOCKED ORDER BY keys are unchanged by
	// the add (the sort rides base's aliases since the R7 P4 wrapper).
	for _, cte := range []string{"jj AS MATERIALIZED", "dd AS MATERIALIZED"} {
		if !strings.Contains(sql, cte) {
			t.Errorf("category add disturbed the MATERIALIZED plan pin %q\nSQL:\n%s", cte, sql)
		}
	}
	if !strings.Contains(sql, "ORDER BY subscription_group_name ASC NULLS FIRST, job_template_name ASC,") {
		t.Errorf("category add changed the LOCKED ORDER BY\nSQL:\n%s", sql)
	}
}

// TestCollateDeliverySummaries pins the multi-deliverer fold (S8 §F): per-(template,
// staff) rows collapse into ONE summary per template carrying ALL deliverers in
// arrival order; single-deliverer templates keep one deliverer; job_count is the
// MAX across a template's rows; a template spanning two delivery groups keeps
// distinct rows (only the staff axis folds).
func TestCollateDeliverySummaries(t *testing.T) {
	t.Run("single_deliverer_passthrough", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", templateName: "Math", groupID: "g1", groupName: "Nickel",
				staffID: "s1", staffName: "Ana", jobCount: 28,
				priceScheduleID: "ps1", priceScheduleName: "AY25-26", outputProductID: "p1", outputProductName: "Math"},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 summary, got %d", len(got))
		}
		if got[0].GetJobCount() != 28 {
			t.Errorf("job_count: want 28, got %d", got[0].GetJobCount())
		}
		if len(got[0].GetDeliverers()) != 1 || got[0].GetDeliverers()[0].GetStaffName() != "Ana" {
			t.Errorf("want single deliverer Ana, got %+v", got[0].GetDeliverers())
		}
	})

	t.Run("two_deliverers_fold_into_one_row_stable_order", func(t *testing.T) {
		// Ordered upstream by staff_name (Cabornay before Purisima).
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", templateName: "Arts — AY 2025-2026", groupID: "g1", groupName: "Nickel",
				staffID: "s2", staffName: "D. Cabornay", jobCount: 28,
				priceScheduleID: "ps1", outputProductID: "p1"},
			{templateID: "t1", templateName: "Arts — AY 2025-2026", groupID: "g1", groupName: "Nickel",
				staffID: "s1", staffName: "A. Purisima", jobCount: 28,
				priceScheduleID: "ps1", outputProductID: "p1"},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 folded summary, got %d", len(got))
		}
		dels := got[0].GetDeliverers()
		if len(dels) != 2 {
			t.Fatalf("want 2 deliverers, got %d", len(dels))
		}
		if dels[0].GetStaffName() != "D. Cabornay" || dels[1].GetStaffName() != "A. Purisima" {
			t.Errorf("arrival order not preserved: %q, %q", dels[0].GetStaffName(), dels[1].GetStaffName())
		}
		if got[0].GetJobCount() != 28 {
			t.Errorf("merged roster job_count: want 28 (MAX), got %d", got[0].GetJobCount())
		}
	})

	t.Run("job_count_is_max_across_rows", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 20},
			{templateID: "t1", groupID: "g1", staffID: "s2", staffName: "B", jobCount: 28},
		})
		if len(got) != 1 || got[0].GetJobCount() != 28 {
			t.Errorf("want single summary with MAX job_count 28, got %+v", got)
		}
	})

	t.Run("distinct_groups_stay_separate", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", groupName: "Nickel", staffID: "s1", staffName: "A", jobCount: 28},
			{templateID: "t1", groupID: "g2", groupName: "Tin", staffID: "s1", staffName: "A", jobCount: 26},
		})
		if len(got) != 2 {
			t.Fatalf("a template spanning two groups must keep 2 rows, got %d", len(got))
		}
	})

	t.Run("blank_staff_id_yields_no_deliverer", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "", staffName: "", jobCount: 0},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 summary, got %d", len(got))
		}
		if len(got[0].GetDeliverers()) != 0 {
			t.Errorf("blank staff must not create a deliverer, got %+v", got[0].GetDeliverers())
		}
	})

	// R9 W-A1: job_category_id rides through the fold. It is functionally determined
	// by the template, so it is set once (first-seen) and stays constant across a
	// key's folded staff rows.
	t.Run("job_category_id_set_once_constant_across_folded_staff", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 28, jobCategoryID: "cat-academic"},
			{templateID: "t1", groupID: "g1", staffID: "s2", staffName: "B", jobCount: 28, jobCategoryID: "cat-academic"},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 folded summary, got %d", len(got))
		}
		if got[0].GetJobCategoryId() != "cat-academic" {
			t.Errorf("job_category_id: want cat-academic (constant across staff rows), got %q", got[0].GetJobCategoryId())
		}
	})

	// R7 P4: BOTH approval quadruples ride the fold set-once — the template-wide
	// quadruple (fields 14-17) is determined by the template, the group+template
	// quadruple (fields 18-21) by (template, group); both ⊆ the collation key, so
	// they are constant across a key's folded staff rows.
	t.Run("approval_quadruples_set_once_across_folded_staff", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 28,
				publishedCount: 1, phaseCount: 2, lowestRank: 1, mixedAttention: true,
				groupPublishedCount: 1, groupPhaseCount: 1, groupLowestRank: 4, groupMixedAttention: false},
			{templateID: "t1", groupID: "g1", staffID: "s2", staffName: "B", jobCount: 28,
				publishedCount: 1, phaseCount: 2, lowestRank: 1, mixedAttention: true,
				groupPublishedCount: 1, groupPhaseCount: 1, groupLowestRank: 4, groupMixedAttention: false},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 folded summary, got %d", len(got))
		}
		s := got[0]
		if s.GetPublishedCount() != 1 || s.GetPhaseCount() != 2 ||
			s.GetLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS ||
			!s.GetMixedAttention() {
			t.Errorf("template-wide quadruple wrong: %+v", s)
		}
		if s.GetGroupPublishedCount() != 1 || s.GetGroupPhaseCount() != 1 ||
			s.GetGroupLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED ||
			s.GetGroupMixedAttention() {
			t.Errorf("group+template quadruple wrong: %+v", s)
		}
	})

	// R7 P4 dual grain: a template spanning two groups keeps DISTINCT rows whose
	// template-wide quadruple repeats but whose GROUP quadruple differs per row —
	// the R9 cell reads per-group state, the courses row template-wide state.
	t.Run("group_grain_differs_across_a_templates_groups", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 10,
				publishedCount: 1, phaseCount: 2, lowestRank: 1, mixedAttention: false,
				groupPublishedCount: 1, groupPhaseCount: 1, groupLowestRank: 4, groupMixedAttention: false},
			{templateID: "t1", groupID: "g2", staffID: "s1", staffName: "A", jobCount: 12,
				publishedCount: 1, phaseCount: 2, lowestRank: 1, mixedAttention: false,
				groupPublishedCount: 0, groupPhaseCount: 1, groupLowestRank: 1, groupMixedAttention: true},
		})
		if len(got) != 2 {
			t.Fatalf("a template spanning two groups must keep 2 rows, got %d", len(got))
		}
		if got[0].GetPublishedCount() != 1 || got[1].GetPublishedCount() != 1 {
			t.Errorf("template-wide quadruple must repeat on both group rows")
		}
		if got[0].GetGroupLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED ||
			got[1].GetGroupLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS ||
			got[0].GetGroupMixedAttention() || !got[1].GetGroupMixedAttention() {
			t.Errorf("group quadruples must stay per-row: g1=%+v g2=%+v", got[0], got[1])
		}
	})

	// R7 P4: zero data-bearing sheets (SQL NULL lowest_rank → scan 0) map to the
	// UNSPECIFIED enum — the neutral not-started default, never a fake IN_PROGRESS.
	t.Run("no_data_bearing_sheets_map_to_unspecified", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 3,
				publishedCount: 0, phaseCount: 0, lowestRank: 0, mixedAttention: false,
				groupPublishedCount: 0, groupPhaseCount: 0, groupLowestRank: 0, groupMixedAttention: false},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 summary, got %d", len(got))
		}
		if got[0].GetLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED ||
			got[0].GetGroupLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED {
			t.Errorf("no-data template must carry UNSPECIFIED at both grains, got %+v", got[0])
		}
		if got[0].GetPhaseCount() != 0 || got[0].GetGroupPhaseCount() != 0 {
			t.Errorf("no-data template must carry zero denominators, got %+v", got[0])
		}
	})

	// A NULL job_category FK scans (sql.NullString) to the read-model empty string —
	// the landing view maps "" to the single Uncategorized bucket (§3.0). Never
	// dropped, never a category id.
	t.Run("null_job_category_maps_to_empty_uncategorized", func(t *testing.T) {
		got := collateDeliverySummaries([]summaryScanRow{
			{templateID: "t1", groupID: "g1", staffID: "s1", staffName: "A", jobCount: 12, jobCategoryID: ""},
		})
		if len(got) != 1 {
			t.Fatalf("want 1 summary, got %d", len(got))
		}
		if got[0].GetJobCategoryId() != "" {
			t.Errorf("NULL/uncategorized template must map to empty string, got %q", got[0].GetJobCategoryId())
		}
	})
}

func TestSummaryRowDeliveryDecodingAndPagination(t *testing.T) {
	t.Run("ordered_sql_arrays_decode_to_one_complete_row", func(t *testing.T) {
		row := summaryFromScanRow(summaryScanRow{
			templateID: "template-1", groupID: "group-1", jobCount: 28,
			deliverers: deliveryPairs([]string{"staff-b", "staff-a"}, []string{"B. Teacher", "A. Teacher"}),
		})
		if got := row.GetDeliverers(); len(got) != 2 || got[0].GetStaffId() != "staff-b" || got[1].GetStaffName() != "A. Teacher" {
			t.Fatalf("ordered delivery decode = %+v, want SQL order preserved", got)
		}
	})

	t.Run("mismatched_sql_arrays_fail_closed", func(t *testing.T) {
		if got := deliveryPairs([]string{"staff-a"}, nil); got != nil {
			t.Fatalf("mismatched delivery arrays = %+v, want nil", got)
		}
	})

	t.Run("same_template_in_two_groups_remains_two_page_units", func(t *testing.T) {
		rows := []*summarypb.JobTemplateSummary{
			{JobTemplateId: "template-1", SubscriptionGroupId: "group-a"},
			{JobTemplateId: "template-1", SubscriptionGroupId: "group-b"},
		}
		page, hasNext := trimSummaryPage(rows, 1)
		if len(page) != 1 || page[0].GetSubscriptionGroupId() != "group-a" || !hasNext {
			t.Fatalf("summary-grain page = %+v, hasNext=%v; want first group only + next", page, hasNext)
		}
	})

	t.Run("limit_plus_one_sets_has_next_only_for_extra_complete_row", func(t *testing.T) {
		one := []*summarypb.JobTemplateSummary{{JobTemplateId: "one"}}
		if page, hasNext := trimSummaryPage(one, 1); len(page) != 1 || hasNext {
			t.Fatalf("exact page = %+v, hasNext=%v; want no next", page, hasNext)
		}
		two := append(one, &summarypb.JobTemplateSummary{JobTemplateId: "two"})
		if page, hasNext := trimSummaryPage(two, 1); len(page) != 1 || !hasNext || page[0].GetJobTemplateId() != "one" {
			t.Fatalf("extra summary row = %+v, hasNext=%v; want first row + next", page, hasNext)
		}
	})
}

// TestJobTemplateSummarySQL_ApprovalPreaggregate locks the page-first approval
// plan: base rows are selected before approval work, then ta and ga are computed
// at their respective template and (template, group) grains without task/outcome
// join fanout.
func TestJobTemplateSummarySQL_ApprovalPreaggregate(t *testing.T) {
	sql := fullSummarySQL()

	t.Run("page_base_precedes_candidate_approval_work", func(t *testing.T) {
		if strings.Contains(sql, ";") {
			t.Errorf("statement must remain a single SQL statement (no ';')\nSQL:\n%s", sql)
		}
		for _, cte := range []string{"page_base AS MATERIALIZED", "candidate_templates AS (", "candidate_groups AS (", "data_phases AS MATERIALIZED", "template_phase AS (", "ta AS MATERIALIZED (", "group_phase AS (", "ga AS MATERIALIZED ("} {
			if !strings.Contains(sql, cte) {
				t.Errorf("missing page-first approval CTE %q\nSQL:\n%s", cte, sql)
			}
		}
		// The page-base estimate can be far lower than its actual row count. These
		// final rollup fences prevent PostgreSQL from inlining and re-running their
		// GroupAggregates once per page row.
		if strings.Contains(sql, "ta AS (") || strings.Contains(sql, "ga AS (") {
			t.Errorf("final approval rollups must remain MATERIALIZED planner fences\nSQL:\n%s", sql)
		}
		if strings.Index(sql, "page_base AS MATERIALIZED") > strings.Index(sql, "candidate_templates AS (") {
			t.Errorf("candidate approvals must follow page_base\nSQL:\n%s", sql)
		}
	})

	t.Run("candidate_grains_and_set_oriented_data_phase_seam", func(t *testing.T) {
		for _, frag := range []string{
			"SELECT DISTINCT job_template_id AS template_id FROM page_base",
			"SELECT DISTINCT job_template_id AS template_id, subscription_group_id AS group_id FROM page_base",
			"FROM candidate_templates ct\n    JOIN jj ON jj.template_id = ct.template_id",
			"FROM candidate_groups cg\n    JOIN jj ON jj.template_id = cg.template_id",
			"gm.subscription_group_id = cg.group_id",
			"gm.subscription_id = jj.subscription_id AND gm.client_id = jj.client_id",
			"data_phases AS MATERIALIZED (\n    SELECT DISTINCT tk.job_phase_id",
			"FROM " + entityid.JobTask + " tk\n    JOIN " + entityid.TaskOutcome + " tox ON tox.job_task_id = tk.id AND tox.active",
			"BOOL_OR(data_phases.job_phase_id IS NOT NULL) AS has_data",
		} {
			if !strings.Contains(sql, frag) {
				t.Errorf("candidate approval CTE missing %q\nSQL:\n%s", frag, sql)
			}
		}
		if n := strings.Count(sql, "LEFT JOIN data_phases ON data_phases.job_phase_id = jp.id"); n != 2 {
			t.Errorf("want exactly two phase-grain joins to data_phases, got %d\nSQL:\n%s", n, sql)
		}
		if strings.Contains(sql, "EXISTS (") {
			t.Errorf("approval phases must not contain correlated EXISTS probes\nSQL:\n%s", sql)
		}
		if n := strings.Count(sql, "FROM "+entityid.JobTask+" tk"); n != 1 {
			t.Errorf("job_task must be scanned once in data_phases, got %d\nSQL:\n%s", n, sql)
		}
		if n := strings.Count(sql, "JOIN "+entityid.TaskOutcome+" tox"); n != 1 {
			t.Errorf("task_outcome must be scanned once in data_phases, got %d\nSQL:\n%s", n, sql)
		}
	})

	t.Run("both_grains_projected_and_joined", func(t *testing.T) {
		// Template-wide (ta, fields 14-17) and group+template (ga, fields 18-21)
		// each project the full quadruple, COALESCEd on the LEFT join.
		for _, col := range []string{
			"COALESCE(ta.published_count, 0)        AS published_count",
			"COALESCE(ta.phase_count, 0)            AS phase_count",
			"ta.lowest_rank                         AS lowest_rank",
			"COALESCE(ta.mixed_attention, false)    AS mixed_attention",
			"COALESCE(ga.published_count, 0)        AS group_published_count",
			"COALESCE(ga.phase_count, 0)            AS group_phase_count",
			"ga.lowest_rank                         AS group_lowest_rank",
			"COALESCE(ga.mixed_attention, false)    AS group_mixed_attention",
		} {
			if !strings.Contains(sql, col) {
				t.Errorf("outer SELECT missing approval column %q\nSQL:\n%s", col, sql)
			}
		}
		if !strings.Contains(sql, "FROM page_base\n    LEFT JOIN ta\n           ON ta.template_id = page_base.job_template_id") {
			t.Errorf("ta must join final page rows on template_id\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "LEFT JOIN ga\n           ON ga.template_id = page_base.job_template_id AND ga.group_id = page_base.subscription_group_id") {
			t.Errorf("ga must join final page rows on (template_id, group_id)\nSQL:\n%s", sql)
		}
	})

	t.Run("no_data_phases_excluded_from_denominator", func(t *testing.T) {
		// D3/Q-R9-1: every ta/ga aggregate filters on has_data — no-data sheets
		// are excluded from published_count, phase_count, lowest_rank AND mixed.
		wantFilters := []struct {
			frag string
			n    int
		}{
			{"COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count", 2}, // ta + ga
			{"COUNT(*) FILTER (WHERE has_data)                  AS phase_count", 2},
			{"MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank", 2},
			{"BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention", 2},
		}
		for _, w := range wantFilters {
			if n := strings.Count(sql, w.frag); n != w.n {
				t.Errorf("want %d occurrences (ta+ga) of %q, got %d\nSQL:\n%s", w.n, w.frag, n, sql)
			}
		}
		if n := strings.Count(sql, "BOOL_OR(data_phases.job_phase_id IS NOT NULL) AS has_data"); n != 2 {
			t.Errorf("want two set-oriented has_data seams (template + group), got %d\nSQL:\n%s", n, sql)
		}
	})

	t.Run("rank_case_mirrors_matrix_rollup", func(t *testing.T) {
		// The exact ladder-rank CASE (unknown → 1, fail-conservative), used for
		// both MIN and MAX at template_phase and group_phase grains.
		for _, tok := range []string{
			"WHEN 'PHASE_APPROVAL_STATUS_IN_PROGRESS' THEN 1",
			"WHEN 'PHASE_APPROVAL_STATUS_FOR_REVIEW'  THEN 2",
			"WHEN 'PHASE_APPROVAL_STATUS_VERIFIED'    THEN 3",
			"WHEN 'PHASE_APPROVAL_STATUS_PUBLISHED'   THEN 4",
		} {
			if n := strings.Count(sql, tok); n != 4 { // MIN + MAX × two grains
				t.Errorf("want rank CASE token %q four times, got %d\nSQL:\n%s", tok, n, sql)
			}
		}
	})

	t.Run("pagination_is_inside_page_base", func(t *testing.T) {
		stmt, _ := buildListJobTemplateSummariesSQL("ws-1", "", "", 25, 0, nil)
		pageStart := strings.Index(stmt, "page_base AS MATERIALIZED")
		approvalStart := strings.Index(stmt, "candidate_templates AS (")
		limit := strings.Index(stmt, "LIMIT $2 OFFSET $3")
		if pageStart < 0 || limit < pageStart || limit > approvalStart {
			t.Errorf("LIMIT must be inside page_base before approval CTEs\nSQL:\n%s", stmt)
		}
	})

	t.Run("wa1_projection_and_order_by_untouched", func(t *testing.T) {
		// W-A1's category projection + GROUP BY tail survive byte-for-byte; the
		// LOCKED ORDER BY keys keep their semantics through base's aliases.
		if !strings.Contains(sql, "jt.job_category_id AS job_category_id") {
			t.Errorf("W-A1 jj category projection disturbed\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "job_category_id, COUNT(DISTINCT job_id) AS job_count") {
			t.Errorf("W-A1 category summary-grain count disturbed\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "ORDER BY subscription_group_name ASC NULLS FIRST, job_template_name ASC,") {
			t.Errorf("preaggregate add changed the LOCKED ORDER BY keys\nSQL:\n%s", sql)
		}
	})

	t.Run("placeholder_numbering_unchanged", func(t *testing.T) {
		// The preaggregate CTEs reference only $1 (workspace) — status/group/
		// scope/limit numbering is untouched.
		stmt, args := buildListJobTemplateSummariesSQL("ws-1", "JOB_STATUS_ACTIVE", "grp-1", 25, 0, nil)
		if !strings.Contains(stmt, "sg.id = $3") || !strings.Contains(stmt, "LIMIT $4 OFFSET $5") {
			t.Errorf("preaggregate add shifted placeholder numbering:\n%s", stmt)
		}
		assertArgs(t, args, []any{"ws-1", "JOB_STATUS_ACTIVE", "grp-1", int32(25), int32(0)})
	})
}

// TestJobTemplateSummaryDB_ApprovalPreaggregate is the READ-ONLY, DB-gated
// P4 evidence run (TEST_DATABASE_URL; optional TEST_STAFF_ID pins the STAFF
// persona). It proves, against the live schema:
//   - grain parity: the preaggregate LEFT joins add ZERO rows — no duplicate
//     (template, group, schedule, product) tuple exists for admin OR
//     STAFF (the join keys are unique per collapsed CTE row);
//   - both grains obey their containment invariants (group ⊆ template);
//   - the no-data denominator exclusion (phase_count counts only data-bearing
//     sheets, cross-checked against an independent direct SQL count);
//   - EXPLAIN (ANALYZE, BUFFERS) captured for the admin statement (budget).
//
// Executes only SELECT/EXPLAIN — zero writes, no locks beyond MVCC reads.
func TestJobTemplateSummaryDB_ApprovalPreaggregate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Workspace: the busiest job workspace (education1: the single live edu ws).
	var ws string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&ws); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	t.Logf("workspace: %s", ws)

	adminCtx := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: ws})

	countRows := func(t *testing.T, stmt string, args []any) (n int, dupes int) {
		t.Helper()
		rows, err := db.QueryContext(ctx, `SELECT COUNT(*), COUNT(*) - COUNT(DISTINCT (job_template_id, subscription_group_id, price_schedule_id, output_product_id)) FROM (`+stmt+`) q`, args...)
		if err != nil {
			t.Fatalf("count query: %v", err)
		}
		defer rows.Close()
		if !rows.Next() {
			t.Fatalf("count query returned no row")
		}
		if err := rows.Scan(&n, &dupes); err != nil {
			t.Fatalf("scan count: %v", err)
		}
		return n, dupes
	}

	adminStmt, adminArgs := buildListJobTemplateSummariesSQL(ws, "", "", 0, 0, nil)

	t.Run("admin_grain_parity", func(t *testing.T) {
		n, dupes := countRows(t, adminStmt, adminArgs)
		t.Logf("admin rows=%d duplicate-grain-tuples=%d", n, dupes)
		if dupes != 0 {
			t.Errorf("preaggregate fanned out the delivery grain: %d duplicate tuples", dupes)
		}
		if n == 0 {
			t.Errorf("admin summary returned zero rows — fixture/workspace mismatch")
		}
	})

	t.Run("adapter_invariants_both_grains", func(t *testing.T) {
		q := NewPostgresJobTemplateSummaryQuery(db)
		resp, err := q.ListJobTemplateSummaries(adminCtx, &summarypb.ListJobTemplateSummariesRequest{})
		if err != nil {
			t.Fatalf("adapter list: %v", err)
		}
		grainDiverges := 0
		for _, s := range resp.GetSummaries() {
			if s.GetGroupPhaseCount() > s.GetPhaseCount() {
				t.Errorf("template %s: group_phase_count %d > phase_count %d", s.GetJobTemplateId(), s.GetGroupPhaseCount(), s.GetPhaseCount())
			}
			// Published counts intentionally have no monotonic relation: a group
			// sheet can be fully published while its template-wide phase is not.
			// Template lowest = MIN over groups ⇒ when the group slice bears data,
			// its lowest can never rank BELOW the template-wide lowest.
			if gr, tr := int(s.GetGroupLowestStatus()), int(s.GetLowestStatus()); gr > 0 && tr > 0 && gr < tr {
				t.Errorf("template %s: group lowest %d ranks below template-wide lowest %d", s.GetJobTemplateId(), gr, tr)
			}
			if s.GetPhaseCount() == 0 && s.GetLowestStatus() != jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED {
				t.Errorf("template %s: zero data-bearing sheets must be UNSPECIFIED, got %v", s.GetJobTemplateId(), s.GetLowestStatus())
			}
			if s.GetGroupPhaseCount() != s.GetPhaseCount() || s.GetGroupPublishedCount() != s.GetPublishedCount() ||
				s.GetGroupLowestStatus() != s.GetLowestStatus() || s.GetGroupMixedAttention() != s.GetMixedAttention() {
				grainDiverges++
			}
		}
		t.Logf("adapter summaries=%d; rows where group grain != template grain: %d", len(resp.GetSummaries()), grainDiverges)
	})

	t.Run("no_data_sheets_excluded_from_denominator", func(t *testing.T) {
		// Independent cross-check: for every template, phase_count must equal the
		// count of its (template, template_phase) sheets bearing >=1 active
		// task_outcome — NOT its total sheet count.
		const crossSQL = `
WITH sheets AS (
    SELECT j.job_template_id AS template_id, jp.template_phase_id,
           BOOL_OR(tox.job_task_id IS NOT NULL) AS has_data
    FROM ` + entityid.Job + ` j
    JOIN ` + entityid.JobPhase + ` jp ON jp.job_id = j.id AND jp.active AND jp.template_phase_id IS NOT NULL
    LEFT JOIN ` + entityid.JobTask + ` tk ON tk.job_phase_id = jp.id AND tk.active
    LEFT JOIN ` + entityid.TaskOutcome + ` tox ON tox.job_task_id = tk.id AND tox.active
    WHERE j.workspace_id = $1 AND j.active AND j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
      AND j.job_template_id IS NOT NULL
    GROUP BY 1, 2
)
SELECT template_id,
       COUNT(*)                            AS total_sheets,
       COUNT(*) FILTER (WHERE has_data)    AS data_sheets
FROM sheets GROUP BY template_id`
		type sheetCounts struct{ total, data int32 }
		expect := map[string]sheetCounts{}
		rows, err := db.QueryContext(ctx, crossSQL, ws)
		if err != nil {
			t.Fatalf("cross-check query: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var c sheetCounts
			if err := rows.Scan(&id, &c.total, &c.data); err != nil {
				t.Fatalf("scan cross-check: %v", err)
			}
			expect[id] = c
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("cross-check rows: %v", err)
		}

		q := NewPostgresJobTemplateSummaryQuery(db)
		resp, err := q.ListJobTemplateSummaries(adminCtx, &summarypb.ListJobTemplateSummariesRequest{})
		if err != nil {
			t.Fatalf("adapter list: %v", err)
		}
		excludedSomewhere := 0
		for _, s := range resp.GetSummaries() {
			c, ok := expect[s.GetJobTemplateId()]
			if !ok {
				continue
			}
			if s.GetPhaseCount() != c.data {
				t.Errorf("template %s: phase_count=%d want data-bearing=%d (total=%d)",
					s.GetJobTemplateId(), s.GetPhaseCount(), c.data, c.total)
			}
			if c.data < c.total {
				excludedSomewhere++
			}
		}
		t.Logf("templates where no-data sheets were excluded (data < total): %d", excludedSomewhere)
	})

	t.Run("staff_grain_parity", func(t *testing.T) {
		staffID := os.Getenv("TEST_STAFF_ID")
		if staffID == "" {
			if err := db.QueryRowContext(ctx,
				`SELECT staff_id FROM `+entityid.SubscriptionSeat+` WHERE status = 'active' AND active AND workspace_id = $1 ORDER BY staff_id LIMIT 1`,
				ws,
			).Scan(&staffID); err != nil {
				t.Skipf("no active seat staff to impersonate: %v", err)
			}
		}
		staffCtx := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
			WorkspaceID: ws, PrincipalType: 7, PrincipalID: staffID,
		})
		// Build the STAFF-shaped statement exactly as the adapter does (the same
		// StaffReachableJobClause splice on the jj CTE's "j" alias).
		scopeFn := func(startParam int) (string, []any) {
			return principalscope.StaffReachableJobClause(staffCtx, "j", startParam)
		}
		stmt, args := buildListJobTemplateSummariesSQL(ws, "", "", 0, 0, scopeFn)
		n, dupes := countRows(t, stmt, args)
		t.Logf("STAFF %s rows=%d duplicate-grain-tuples=%d", staffID, n, dupes)
		if dupes != 0 {
			t.Errorf("STAFF preaggregate fanned out the delivery grain: %d duplicate tuples", dupes)
		}

		q := NewPostgresJobTemplateSummaryQuery(db)
		resp, err := q.ListJobTemplateSummaries(staffCtx, &summarypb.ListJobTemplateSummariesRequest{})
		if err != nil {
			t.Fatalf("adapter STAFF list: %v", err)
		}
		t.Logf("STAFF adapter summaries=%d", len(resp.GetSummaries()))

		// Budget evidence: the STAFF-scoped statement's live timing.
		erows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE) "+stmt, args...)
		if err != nil {
			t.Fatalf("staff explain: %v", err)
		}
		defer erows.Close()
		for erows.Next() {
			var line string
			if err := erows.Scan(&line); err != nil {
				t.Fatalf("scan staff plan line: %v", err)
			}
			if strings.Contains(line, "Execution Time") || strings.Contains(line, "Planning Time") {
				t.Logf("STAFF %s", strings.TrimSpace(line))
			}
		}
		if err := erows.Err(); err != nil {
			t.Fatalf("staff plan rows: %v", err)
		}
	})

	t.Run("explain_analyze_admin", func(t *testing.T) {
		rows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+adminStmt, adminArgs...)
		if err != nil {
			t.Fatalf("explain: %v", err)
		}
		defer rows.Close()
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("scan plan line: %v", err)
			}
			plan.WriteString(line)
			plan.WriteString("\n")
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("plan rows: %v", err)
		}
		t.Logf("EXPLAIN (ANALYZE, BUFFERS) admin statement:\n%s", plan.String())
	})
}

// TestJobTemplateSummaryDB_ExplainShapes records non-executing planner shapes
// for the complete compatibility request and the bounded first page. It is a
// cheap diagnostic companion to the ANALYZE gate above: a pathological plan can
// be inspected without waiting for its execution or weakening read-only guards.
func TestJobTemplateSummaryDB_ExplainShapes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}

	for _, tc := range []struct {
		name  string
		limit int32
	}{
		{name: "unpaginated"},
		{name: "first_page_100", limit: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, args := buildListJobTemplateSummariesSQL(workspaceID, "", "", tc.limit, 0, nil)
			rows, err := db.QueryContext(ctx, "EXPLAIN (VERBOSE, COSTS, FORMAT TEXT) "+stmt, args...)
			if err != nil {
				t.Fatalf("EXPLAIN shape: %v", err)
			}
			defer rows.Close()
			var plan strings.Builder
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					t.Fatalf("scan EXPLAIN shape: %v", err)
				}
				plan.WriteString(line)
				plan.WriteByte('\n')
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("EXPLAIN shape rows: %v", err)
			}
			t.Logf("%s planner shape:\n%s", tc.name, plan.String())
		})
	}
}

// TestJobTemplateSummaryDB_PagedExplain captures the actual plan for the
// status-filtered first Courses page. It is gated by TEST_DATABASE_URL; callers
// must supply PGOPTIONS with a read-only transaction and suitable timeouts.
func TestJobTemplateSummaryDB_PagedExplain(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}

	stmt, args := buildListJobTemplateSummariesSQL(workspaceID, "JOB_STATUS_ACTIVE", "", 100, 0, nil)
	rows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+stmt, args...)
	if err != nil {
		t.Fatalf("paged EXPLAIN: %v", err)
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan paged EXPLAIN line: %v", err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("paged EXPLAIN rows: %v", err)
	}
	t.Logf("paged Courses EXPLAIN (ANALYZE, BUFFERS):\n%s", plan.String())
}

// TestJobTemplateSummaryDB_PagedPerformance is the PERF-01 acceptance probe:
// one bounded page must contain complete response-grain rows, expose truthful
// next-page metadata, and finish inside the read-only session budget. Absolute
// timing is logged as development evidence rather than asserted as a flaky unit
// threshold; the plan artifact is compared with the frozen baseline.
func TestJobTemplateSummaryDB_PagedPerformance(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspaceID})
	req := &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE",
		Pagination: &commonpb.PaginationRequest{
			Limit: 100,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 1},
			},
		},
	}

	started := time.Now()
	resp, err := NewPostgresJobTemplateSummaryQuery(db).ListJobTemplateSummaries(ctx, req)
	if err != nil {
		t.Fatalf("bounded Courses page: %v", err)
	}
	elapsed := time.Since(started)
	if len(resp.GetSummaries()) != 100 {
		t.Fatalf("bounded Courses page rows = %d, want 100", len(resp.GetSummaries()))
	}
	if resp.GetPagination() == nil || !resp.GetPagination().GetHasNext() || resp.GetPagination().GetCurrentPage() != 1 {
		t.Fatalf("bounded Courses pagination = %+v, want page 1 with has_next", resp.GetPagination())
	}
	t.Logf("bounded Courses page: rows=%d elapsed=%s", len(resp.GetSummaries()), elapsed)
}

// TestJobTemplateSummaryDB_ServerPageMetadata proves the one-round-trip
// metadata contract against live data, including zero-row and out-of-range
// pages where a window-only count would disappear. It is SELECT-only and uses
// the same read-only TEST_DATABASE_URL gate as the performance probe.
func TestJobTemplateSummaryDB_ServerPageMetadata(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspaceID})
	query := NewPostgresJobTemplateSummaryQuery(db)
	page := func(number int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{
			Limit: 25,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: number},
			},
		}
	}
	call := func(t *testing.T, req *summarypb.ListJobTemplateSummariesRequest) *summarypb.ListJobTemplateSummariesResponse {
		t.Helper()
		resp, err := query.ListJobTemplateSummaries(ctx, req)
		if err != nil {
			t.Fatalf("ListJobTemplateSummaries: %v", err)
		}
		return resp
	}
	countsMap := func(rows []*summarypb.JobCategorySummaryCount) map[string]int32 {
		out := make(map[string]int32, len(rows))
		for _, row := range rows {
			out[row.GetJobCategoryId()] = row.GetSummaryCount()
		}
		return out
	}

	base := call(t, &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE", Pagination: page(1),
	})
	if base.GetPagination() == nil || base.GetPagination().GetTotalItems() <= 25 || base.GetPagination().GetTotalPages() <= 1 {
		t.Fatalf("base pagination = %+v, want exact multi-page totals", base.GetPagination())
	}
	baseCounts := countsMap(base.GetJobCategoryCounts())
	var countSum int32
	var selectedCategory string
	for categoryID, count := range baseCounts {
		countSum += count
		if selectedCategory == "" && count > 0 {
			selectedCategory = categoryID
		}
	}
	if countSum != base.GetPagination().GetTotalItems() {
		t.Fatalf("category count sum = %d, total_items = %d", countSum, base.GetPagination().GetTotalItems())
	}
	if selectedCategory == "" {
		t.Fatal("base response has no populated category")
	}

	selected := selectedCategory
	categoryResp := call(t, &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE", JobCategoryId: &selected, Pagination: page(1),
	})
	if got, want := categoryResp.GetPagination().GetTotalItems(), baseCounts[selectedCategory]; got != want {
		t.Fatalf("selected category total_items = %d, want badge count %d", got, want)
	}
	if got := countsMap(categoryResp.GetJobCategoryCounts()); len(got) != len(baseCounts) {
		t.Fatalf("selected category badges = %v, want status-universe %v", got, baseCounts)
	} else {
		for categoryID, want := range baseCounts {
			if got[categoryID] != want {
				t.Fatalf("selected category badge %q = %d, want %d", categoryID, got[categoryID], want)
			}
		}
	}
	for _, summary := range categoryResp.GetSummaries() {
		if summary.GetJobCategoryId() != selectedCategory {
			t.Fatalf("selected category returned %q row", summary.GetJobCategoryId())
		}
	}

	noMatch := call(t, &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE", Pagination: page(1),
		Search: &commonpb.SearchRequest{Query: "__ichizen_summary_no_match_7f80c2__"},
	})
	if len(noMatch.GetSummaries()) != 0 || noMatch.GetPagination().GetTotalItems() != 0 {
		t.Fatalf("no-match response rows=%d pagination=%+v", len(noMatch.GetSummaries()), noMatch.GetPagination())
	}
	if got := countsMap(noMatch.GetJobCategoryCounts()); len(got) != len(baseCounts) {
		t.Fatalf("no-match badges = %v, want status-universe %v", got, baseCounts)
	}

	outOfRangePage := base.GetPagination().GetTotalPages() + 2
	outOfRange := call(t, &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE", Pagination: page(outOfRangePage),
	})
	if len(outOfRange.GetSummaries()) != 0 || outOfRange.GetPagination().GetTotalItems() != base.GetPagination().GetTotalItems() {
		t.Fatalf("out-of-range rows=%d pagination=%+v, want empty rows with retained total %d",
			len(outOfRange.GetSummaries()), outOfRange.GetPagination(), base.GetPagination().GetTotalItems())
	}

	includeFallback := true
	fallback := call(t, &summarypb.ListJobTemplateSummariesRequest{
		Status: "JOB_STATUS_ACTIVE", IncludeTemplateFallback: &includeFallback, Pagination: page(1),
	})
	for _, summary := range fallback.GetSummaries() {
		if !summary.GetTemplateGrainFallback() {
			continue
		}
		if summary.GetSubscriptionGroupId() != "" || summary.GetJobCount() != 0 || len(summary.GetDeliverers()) != 0 {
			t.Fatalf("fallback row leaked delivery grain: %+v", summary)
		}
	}
}

// TestJobTemplateSummaryDB_ComponentTimings uses projection pruning to isolate
// which page components dominate the source-owned query. It is diagnostic only:
// every subtest executes the same bounded statement and changes only the outer
// aggregate that forces delivery arrays and/or approval columns to be evaluated.
func TestJobTemplateSummaryDB_ComponentTimings(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	var workspaceID string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1`,
	).Scan(&workspaceID); err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	stmt, args := buildListJobTemplateSummariesSQL(workspaceID, "JOB_STATUS_ACTIVE", "", 100, 0, nil)

	for _, tc := range []struct {
		name       string
		projection string
	}{
		{name: "row_grain", projection: "COUNT(*)"},
		{name: "delivery_arrays", projection: "COALESCE(SUM(cardinality(staff_ids) + cardinality(staff_names)), 0)"},
		{name: "approvals", projection: "COALESCE(SUM(phase_count + group_phase_count), 0)"},
		{name: "arrays_and_approvals", projection: "COALESCE(SUM(cardinality(staff_ids) + phase_count + group_phase_count), 0)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := time.Now()
			var value int64
			if err := db.QueryRowContext(ctx, "SELECT "+tc.projection+" FROM ("+stmt+") q", args...).Scan(&value); err != nil {
				t.Fatalf("component query: %v", err)
			}
			t.Logf("component=%s value=%d elapsed=%s", tc.name, value, time.Since(started))
		})
	}
}

func assertArgs(t *testing.T, got, want []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args length: want %d, got %d (got=%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d]: want %v (%T), got %v (%T)", i, want[i], want[i], got[i], got[i])
		}
	}
}
