//go:build postgresql

package operation

import (
	"context"
	"os"
	"strings"
	"testing"

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
		{"sgm", entityid.SubscriptionGroupMember}, // dd CTE
		{"sg", entityid.SubscriptionGroup},        // outer
		{"ps", entityid.PriceSchedule},            // outer
		{"ss", entityid.SubscriptionSeat},         // dd CTE
		{"pl", entityid.ProductPlan},              // dd CTE
		{"st", entityid.Staff},                    // dd CTE
		{"op", entityid.Product},                  // outer
		{"jp", entityid.JobPhase},                 // pa CTE (R7 P4)
		{"tk", entityid.JobTask},                  // pa CTE (R7 P4)
		{"tox", entityid.TaskOutcome},             // pa CTE (R7 P4)
		{"gm", entityid.SubscriptionGroupMember},  // pa CTE (R7 P4)
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

	// The over-count trap (subscription_group_product_plan_staff, resolved by
	// (plan,staff) instead of the job's own membership) must NEVER appear.
	if strings.Contains(sql, entityid.SubscriptionGroupProductPlanStaff) {
		t.Errorf("SELECT/FROM leaked the sgpps table %q (cohort-grain over-count trap)\nSQL:\n%s",
			entityid.SubscriptionGroupProductPlanStaff, sql)
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
		// COUNT(DISTINCT) now counts the jj CTE's job_id (was j.id pre-rewrite).
		if !strings.Contains(stmt, "COUNT(DISTINCT jj.job_id)") {
			t.Errorf("DISTINCT job count missing:\n%s", stmt)
		}
		// The LOCKED view order (group name, template name) — since R7 P4 the
		// sort sits ABOVE the base wrapper, keyed through base's output aliases
		// (same key semantics as the pre-P4 `sg.name, jj.template_name`).
		if !strings.Contains(stmt, "ORDER BY base.subscription_group_name, base.job_template_name") {
			t.Errorf("locked ORDER BY (group, template) prefix missing:\n%s", stmt)
		}
		// Deterministic DELIVERER order: the per-template staff rows must be
		// ordered so collateDeliverySummaries folds them in a stable sequence.
		if !strings.Contains(stmt, "ORDER BY base.subscription_group_name, base.job_template_name, base.staff_name, base.staff_id") {
			t.Errorf("deterministic deliverer ORDER BY tail (staff_name, staff_id) missing:\n%s", stmt)
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
	// Carried through the outer SELECT AND appended to GROUP BY — both jj-qualified,
	// so jj.job_category_id must appear at least twice.
	if n := strings.Count(sql, "jj.job_category_id"); n < 2 {
		t.Errorf("want jj.job_category_id in BOTH the outer SELECT and GROUP BY (count=%d)\nSQL:\n%s", n, sql)
	}
	// GROUP BY specifically carries the category on the op.name tail.
	if !strings.Contains(sql, "op.name, jj.job_category_id") {
		t.Errorf("GROUP BY missing jj.job_category_id (must ride the op.name tail)\nSQL:\n%s", sql)
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
	if !strings.Contains(sql, "ORDER BY base.subscription_group_name, base.job_template_name, base.staff_name, base.staff_id") {
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

// TestJobTemplateSummarySQL_ApprovalPreaggregate locks the R7 P4 courses-chip
// preaggregate (plan 20260718-phase-approval-workflow §4.5 + the 20260719-
// report-cards-landing §3.5 grain amendment): statement 2 computes BOTH the
// TEMPLATE-WIDE roll-up (ta — proto fields 14-17, the courses row) AND the
// GROUP+TEMPLATE roll-up (ga — proto fields 18-21, the R9 Phase-B cell) in ONE
// statement; no-data sheets are EXCLUDED from every aggregate (D3/Q-R9-1);
// raw job_phase/job_task/task_outcome rows are confined to the pa CTE (never
// the delivery aggregate); the W-A1 category projection, the MATERIALIZED pins
// and the LOCKED ORDER BY are untouched.
func TestJobTemplateSummarySQL_ApprovalPreaggregate(t *testing.T) {
	sql := fullSummarySQL()

	t.Run("single_statement_with_all_preaggregate_ctes", func(t *testing.T) {
		// Still ONE SQL statement (the courses page keeps statement count == 2:
		// tab-support + this) — no statement separator anywhere.
		if strings.Contains(sql, ";") {
			t.Errorf("statement must remain a single SQL statement (no ';')\nSQL:\n%s", sql)
		}
		for _, cte := range []string{"pa AS (", "tp AS (", "ta AS (", "ga AS ("} {
			if !strings.Contains(sql, cte) {
				t.Errorf("missing approval preaggregate CTE %q\nSQL:\n%s", cte, sql)
			}
		}
		// jj/dd MATERIALIZED pins survive (P1 plan pin untouched).
		for _, cte := range []string{"jj AS MATERIALIZED", "dd AS MATERIALIZED"} {
			if !strings.Contains(sql, cte) {
				t.Errorf("preaggregate add disturbed the MATERIALIZED plan pin %q\nSQL:\n%s", cte, sql)
			}
		}
	})

	t.Run("sourced_from_scoped_jj", func(t *testing.T) {
		// pa MUST read FROM jj (the scoped job set incl. the STAFF splice), so
		// STAFF and admin see a chip over the same job scope as the row.
		if n := strings.Count(sql, "FROM jj"); n < 2 {
			t.Errorf("want pa sourced FROM jj in addition to the outer SELECT (FROM jj count=%d)\nSQL:\n%s", n, sql)
		}
		// The pa join chain: phases of jj's jobs, active + template-backed only.
		for _, frag := range []string{
			"jp.job_id = jj.job_id AND jp.active AND jp.template_phase_id IS NOT NULL",
			"tox.job_task_id = tk.id AND tox.active",
			"tk.job_phase_id = jp.id AND tk.active",
			"gm.subscription_id = jj.subscription_id AND gm.client_id = jj.client_id",
		} {
			if !strings.Contains(sql, frag) {
				t.Errorf("pa CTE missing fragment %q\nSQL:\n%s", frag, sql)
			}
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
		// Joined ONLY AFTER the delivery aggregation (codex-tandem: "join that
		// one-row/template result only after delivery aggregation") — the base
		// wrapper carries the finished delivery grain; ta keys by the template,
		// ga by (template, THIS row's group). Never raw job_phase in the
		// delivery join.
		if !strings.Contains(sql, "base AS (") {
			t.Errorf("delivery aggregation must be wrapped as the base CTE\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "LEFT JOIN ta\n       ON ta.template_id = base.job_template_id") {
			t.Errorf("ta must LEFT JOIN base on template_id AFTER aggregation\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "LEFT JOIN ga\n       ON ga.template_id = base.job_template_id AND ga.group_id = base.subscription_group_id") {
			t.Errorf("ga must LEFT JOIN base on (template_id, group_id) AFTER aggregation\nSQL:\n%s", sql)
		}
		// The pre-aggregation delivery FROM must NOT touch ta/ga (join-after
		// contract): base's GROUP BY still ends at the W-A1 category tail.
		if strings.Contains(sql, "GROUP BY jj.template_id") && !strings.Contains(sql,
			"ps.id, ps.name, jj.output_product_id, op.name, jj.job_category_id\n)") {
			t.Errorf("base GROUP BY must end at the W-A1 12-key tail (no ta/ga keys)\nSQL:\n%s", sql)
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
		// has_data comes from the matrix roll-up's exact task_outcome seam.
		if !strings.Contains(sql, "BOOL_OR(tox.job_task_id IS NOT NULL) AS has_data") {
			t.Errorf("pa missing the has_data outcome seam\nSQL:\n%s", sql)
		}
	})

	t.Run("rank_case_mirrors_matrix_rollup", func(t *testing.T) {
		// The exact ladder-rank CASE (unknown → 1, fail-conservative), used for
		// both MIN and MAX at the pa grain.
		for _, tok := range []string{
			"WHEN 'PHASE_APPROVAL_STATUS_IN_PROGRESS' THEN 1",
			"WHEN 'PHASE_APPROVAL_STATUS_FOR_REVIEW'  THEN 2",
			"WHEN 'PHASE_APPROVAL_STATUS_VERIFIED'    THEN 3",
			"WHEN 'PHASE_APPROVAL_STATUS_PUBLISHED'   THEN 4",
		} {
			if n := strings.Count(sql, tok); n != 2 { // MIN + MAX
				t.Errorf("want rank CASE token %q twice (MIN+MAX), got %d\nSQL:\n%s", tok, n, sql)
			}
		}
	})

	t.Run("group_collapse_before_template_rollup", func(t *testing.T) {
		// tp collapses pa across groups to the R7 sheet grain (ALL rows of one
		// (template, template_phase)) BEFORE ta rolls to one row/template — a
		// multi-group sheet's cross-group divergence stays sheet-mixed and a job
		// in 2 groups cannot double-count a phase.
		if !strings.Contains(sql, "GROUP BY template_id, template_phase_id") {
			t.Errorf("tp must collapse pa to (template, template_phase)\nSQL:\n%s", sql)
		}
		// ga keeps the group axis and drops only ungrouped (NULL) slices.
		if !strings.Contains(sql, "WHERE group_id IS NOT NULL") {
			t.Errorf("ga must exclude the NULL (ungrouped) slice\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "GROUP BY template_id, group_id") {
			t.Errorf("ga must roll to (template, group)\nSQL:\n%s", sql)
		}
	})

	t.Run("wa1_projection_and_order_by_untouched", func(t *testing.T) {
		// W-A1's category projection + GROUP BY tail survive byte-for-byte; the
		// LOCKED ORDER BY keys keep their semantics through base's aliases.
		if !strings.Contains(sql, "jt.job_category_id AS job_category_id") {
			t.Errorf("W-A1 jj category projection disturbed\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "op.name, jj.job_category_id") {
			t.Errorf("W-A1 GROUP BY category tail disturbed\nSQL:\n%s", sql)
		}
		if !strings.Contains(sql, "ORDER BY base.subscription_group_name, base.job_template_name, base.staff_name, base.staff_id") {
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
//     (template, group, staff, schedule, product) tuple exists for admin OR
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
		rows, err := db.QueryContext(ctx, `SELECT COUNT(*), COUNT(*) - COUNT(DISTINCT (job_template_id, subscription_group_id, staff_id, price_schedule_id, output_product_id)) FROM (`+stmt+`) q`, args...)
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
			if s.GetGroupPublishedCount() > s.GetPublishedCount() {
				t.Errorf("template %s: group_published_count %d > published_count %d", s.GetJobTemplateId(), s.GetGroupPublishedCount(), s.GetPublishedCount())
			}
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
