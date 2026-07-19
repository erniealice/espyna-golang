//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
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
	// other workspace-bearing joined table also carries $1 (across both CTEs and
	// the outer SELECT).
	boundAliases := []string{"j", "jt", "sgm", "sg", "ps", "ss", "st", "op"}
	for _, a := range boundAliases {
		if !strings.Contains(sql, a+".workspace_id = $1") {
			t.Errorf("table alias %q missing workspace_id = $1 bind\nSQL:\n%s", a, sql)
		}
	}

	// product_plan (pl) and user (u) have no workspace_id column — a bind there
	// would be a schema error.
	for _, banned := range []string{"pl.workspace_id", "u.workspace_id"} {
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
		if !strings.Contains(stmt, "ORDER BY sg.name, jj.template_name") {
			t.Errorf("locked ORDER BY (group, template) prefix missing:\n%s", stmt)
		}
		// Deterministic DELIVERER order: the per-template staff rows must be
		// ordered so collateDeliverySummaries folds them in a stable sequence.
		if !strings.Contains(stmt, "ORDER BY sg.name, jj.template_name, staff_name, dd.staff_id") {
			t.Errorf("deterministic deliverer ORDER BY tail (staff_name, dd.staff_id) missing:\n%s", stmt)
		}
	})
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
