//go:build postgresql

package principalscope

import (
	"database/sql"
	"fmt"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"slices"
	"testing"
)

// Frozen pre-P2 five-arm set query. Keep independent of the production builder:
// live parity uses this as the old authorization oracle (plan Q6).
func legacyReachableJobUnion(staffP, wsP int) string {
	s := fmt.Sprintf("$%d", staffP)
	w := fmt.Sprintf("$%d", wsP)
	return "SELECT jp.job_id FROM " + entityid.JobPhase + " jp" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id AND jt.workspace_id = " + w +
		" JOIN " + entityid.Job + " jw ON jw.id = jp.job_id AND jw.workspace_id = " + w +
		" WHERE jp.workspace_id = " + w + " AND jt.assigned_to = " + s +
		" UNION " +
		"SELECT jp2.job_id FROM " + entityid.JobPhase + " jp2" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id AND jt2.workspace_id = " + w +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id AND t.workspace_id = " + w +
		" JOIN " + entityid.Job + " jw2 ON jw2.id = jp2.job_id AND jw2.workspace_id = " + w +
		" WHERE jp2.workspace_id = " + w + " AND (t.recorded_by = " + s + " OR t.reviewed_by = " + s + ")" +
		" UNION " +
		"SELECT jw3.id FROM " + entityid.Job + " jw3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = jw3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = jw3.job_template_id AND tpl.workspace_id = " + w +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = " + s + " AND ss.status = 'active' AND ss.active = true" +
		" AND jw3.origin_type = '" + originTypeSubscription + "'" +
		" AND jw3.workspace_id = " + w + " AND ss.workspace_id = " + w +
		" UNION " +
		// Class-edge tier (OPTIMIZED, 4 tables + a v2 eligibility LEFT JOIN): the
		// acting staff is the class-edge servicer (sgpps) for the job's subject. NO
		// job_template hop — job.output_product_id is populated by spawn, so the
		// subject matches on the job directly; NO subscription_group / subscription
		// hops — subscription_group_member carries BOTH subscription_group_id AND
		// subscription_id. Drives from the staff's OWN sgpps edges (idx_..._staff_id,
		// a tiny starting set) and is fail-closed identically to the tiers above: an
		// empty staff/workspace bind matches no sgpps row (both filters land on the
		// edge), so it yields zero rows and cannot be widened by a request param.
		// Staff resolution is v2-native: COALESCE(pps.staff_id, e.staff_id) prefers
		// the edge's linked product_plan_staff eligibility row (f13) and falls back
		// to the edge's own legacy staff_id (f10) only when unlinked — see the
		// COALESCE note on this function's doc comment. classEdgeEligibilityLive
		// then retracts a LINKED-but-REVOKED eligibility (see its own comment: the
		// fallback must NOT rescue such a row).
		"SELECT jce.id FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.workspace_id = " + w + " AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id AND jce.workspace_id = " + w +
		" LEFT JOIN " + entityid.ProductPlanStaff + " pps ON pps.id = e.product_plan_staff_id AND pps.workspace_id = " + w +
		" WHERE COALESCE(pps.staff_id, e.staff_id) = " + s + classEdgeEligibilityLive +
		" AND e.active AND e.workspace_id = " + w +
		" UNION " +
		// Reviewer tier (20260924-approval-role-workflow D3).
		reviewerTierSQL("jr.id", s, w, "")
}

func reachableOrderedIDs(t *testing.T, db *sql.DB, clause string, staff, workspace string) []string {
	t.Helper()
	rows, err := db.Query("SELECT j.id FROM "+entityid.Job+" j WHERE j.workspace_id = $2 "+clause+" ORDER BY j.id", staff, workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

// Each selector is constrained by the same joins as its reachability arm, so a
// sampled staff member must have at least one old-union result. A missing cohort
// is explicitly skipped and remains an evidence gap for this clone.
func TestReachableJobDB_Parity(t *testing.T) {
	db := openClassEdgeV2LiveDB(t)
	defer db.Close()
	cohorts := []struct{ name, sample string }{
		{"assignee", `SELECT jt.assigned_to, jp.workspace_id FROM job_task jt JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=jt.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=jp.workspace_id WHERE jt.assigned_to IS NOT NULL AND jt.assigned_to<>'' ORDER BY jt.assigned_to LIMIT 1`},
		{"recorder", `SELECT t.recorded_by, t.workspace_id FROM task_outcome t JOIN job_task jt ON jt.id=t.job_task_id AND jt.workspace_id=t.workspace_id JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=t.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=t.workspace_id WHERE t.recorded_by IS NOT NULL AND t.recorded_by<>'' ORDER BY t.recorded_by LIMIT 1`},
		{"approver", `SELECT t.reviewed_by, t.workspace_id FROM task_outcome t JOIN job_task jt ON jt.id=t.job_task_id AND jt.workspace_id=t.workspace_id JOIN job_phase jp ON jp.id=jt.job_phase_id AND jp.workspace_id=t.workspace_id JOIN job j ON j.id=jp.job_id AND j.workspace_id=t.workspace_id WHERE t.reviewed_by IS NOT NULL AND t.reviewed_by<>'' ORDER BY t.reviewed_by LIMIT 1`},
		{"subscription_seat", `SELECT ss.staff_id, ss.workspace_id FROM subscription_seat ss JOIN job j ON j.origin_id=ss.subscription_id AND j.workspace_id=ss.workspace_id AND j.origin_type='ORIGIN_TYPE_SUBSCRIPTION' JOIN job_template tpl ON tpl.id=j.job_template_id AND tpl.workspace_id=ss.workspace_id JOIN product_plan pl ON pl.id=ss.product_plan_id AND pl.product_id=tpl.output_product_id WHERE ss.staff_id IS NOT NULL AND ss.status='active' AND ss.active ORDER BY ss.staff_id LIMIT 1`},
		{"class_edge", `SELECT COALESCE(pps.staff_id,e.staff_id),e.workspace_id FROM subscription_group_product_plan_staff e JOIN subscription_group_member m ON m.subscription_group_id=e.subscription_group_id AND m.workspace_id=e.workspace_id AND m.active JOIN product_plan pp ON pp.id=e.product_plan_id JOIN job jce ON jce.origin_id=m.subscription_id AND jce.output_product_id=pp.product_id AND jce.workspace_id=e.workspace_id LEFT JOIN product_plan_staff pps ON pps.id=e.product_plan_staff_id AND pps.workspace_id=e.workspace_id WHERE e.active AND (e.product_plan_staff_id IS NULL OR pps.active) AND COALESCE(pps.staff_id,e.staff_id) IS NOT NULL ORDER BY COALESCE(pps.staff_id,e.staff_id) LIMIT 1`},
		{"reviewer", `SELECT r.staff_id,r.workspace_id FROM product_plan_staff r JOIN subscription_group_product_plan rc ON rc.product_plan_id=r.product_plan_id AND rc.active AND rc.workspace_id=r.workspace_id JOIN subscription_group_member rm ON rm.subscription_group_id=rc.subscription_group_id AND rm.active AND rm.workspace_id=r.workspace_id JOIN product_plan rpp ON rpp.id=r.product_plan_id JOIN job jr ON jr.origin_id=rm.subscription_id AND jr.output_product_id=rpp.product_id AND jr.workspace_id=r.workspace_id WHERE r.staff_id IS NOT NULL AND r.role='reviewer' AND r.active ORDER BY r.staff_id LIMIT 1`},
	}
	legacy := "AND j.id IN (" + legacyReachableJobUnion(1, 2) + ")"
	candidate := "AND (" + reachableJobPredicate("j", 1, 2) + ")"
	for _, cohort := range cohorts {
		t.Run(cohort.name, func(t *testing.T) {
			var staff, workspace string
			if err := db.QueryRow(cohort.sample).Scan(&staff, &workspace); err == sql.ErrNoRows {
				t.Skip("no sampled staff in this cohort")
			} else if err != nil {
				t.Fatal(err)
			}
			want := reachableOrderedIDs(t, db, legacy, staff, workspace)
			got := reachableOrderedIDs(t, db, candidate, staff, workspace)
			if len(want) == 0 || !slices.Equal(got, want) {
				t.Fatalf("cohort %s: ordered job sets differ: old=%d new=%d", cohort.name, len(want), len(got))
			}
			t.Logf("ordered jobs=%d", len(got))
		})
	}
	t.Run("empty_staff", func(t *testing.T) {
		var workspace string
		if err := db.QueryRow("SELECT workspace_id FROM " + entityid.Job + " ORDER BY workspace_id LIMIT 1").Scan(&workspace); err != nil {
			t.Fatal(err)
		}
		if got := reachableOrderedIDs(t, db, candidate, "", workspace); len(got) != 0 {
			t.Fatalf("empty staff reached %d jobs", len(got))
		}
	})
}
