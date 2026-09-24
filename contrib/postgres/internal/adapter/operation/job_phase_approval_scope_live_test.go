//go:build postgresql

package operation

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// TestReviewerScopeLive proves the reviewer-edge scope probe against a real
// database (plan 20260924-approval-role-workflow D3): a staff member with no
// reviewer row is OUTSIDE a section-narrowed sheet; after a reviewer row on the
// sheet's product plan offering they are INSIDE. Everything runs in ONE
// transaction that is always rolled back — nothing persists. Skips without
// TEST_DATABASE_URL, or when the database has no suitable sheet.
func TestReviewerScopeLive(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping live reviewer-scope test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck // always rolled back by design

	// One current-section, class-model sheet: template + template phase + section
	// + the class's product plan offering + workspace.
	var templateID, phaseID, groupID, productPlanID, wsID string
	err = tx.QueryRow(`
		SELECT j.job_template_id, jp.template_phase_id, m.subscription_group_id, c.product_plan_id, j.workspace_id
		FROM ` + entityid.Job + ` j
		JOIN ` + entityid.JobPhase + ` jp ON jp.job_id = j.id AND jp.active
		JOIN ` + entityid.SubscriptionGroupMember + ` m ON m.client_id = j.client_id AND m.subscription_id = j.origin_id AND m.active AND m.workspace_id = j.workspace_id
		JOIN ` + entityid.SubscriptionGroup + ` sg ON sg.id = m.subscription_group_id AND sg.status = 'current'
		JOIN ` + entityid.SubscriptionGroupProductPlan + ` c ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = j.workspace_id
		JOIN ` + entityid.ProductPlan + ` pp ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id
		WHERE j.output_product_id IS NOT NULL AND jp.template_phase_id IS NOT NULL
		LIMIT 1`).Scan(&templateID, &phaseID, &groupID, &productPlanID, &wsID)
	if err == sql.ErrNoRows {
		t.Skip("no class-model sheet on this database")
	}
	if err != nil {
		t.Fatalf("pick sheet: %v", err)
	}

	// A staff member of that workspace with NO product_plan_staff row on the offering.
	var staffID string
	err = tx.QueryRow(`
		SELECT s.id FROM `+entityid.Staff+` s
		WHERE s.workspace_id = $1 AND s.active
		  AND NOT EXISTS (SELECT 1 FROM `+entityid.ProductPlanStaff+` p WHERE p.staff_id = s.id AND p.product_plan_id = $2)
		LIMIT 1`, wsID, productPlanID).Scan(&staffID)
	if err == sql.ErrNoRows {
		t.Skip("no staff without a row on the offering")
	}
	if err != nil {
		t.Fatalf("pick staff: %v", err)
	}

	narrow, narrowArgs := groupNarrowPredicate(groupID, 5, 3)
	probe := sheetOutsideReviewerScopeSQL(narrow)
	outside := func() bool {
		t.Helper()
		var v bool
		if err := tx.QueryRow(probe, append([]any{templateID, phaseID, wsID, staffID}, narrowArgs...)...).Scan(&v); err != nil {
			t.Fatalf("probe: %v", err)
		}
		return v
	}

	if !outside() {
		t.Fatal("staff with no reviewer row must be OUTSIDE the sheet's approval scope")
	}

	if _, err := tx.Exec(`INSERT INTO `+entityid.ProductPlanStaff+`
		(id, date_created, date_modified, active, workspace_id, product_plan_id, staff_id, role)
		VALUES ('live-test-reviewer-edge', 0, 0, true, $1, $2, $3, 'reviewer')`, wsID, productPlanID, staffID); err != nil {
		t.Fatalf("insert reviewer row (rolled back): %v", err)
	}
	if outside() {
		t.Fatal("staff with a reviewer row on the sheet's offering must be INSIDE the approval scope")
	}

	if _, err := tx.Exec(`UPDATE `+entityid.ProductPlanStaff+` SET active = false WHERE id = 'live-test-reviewer-edge'`); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if !outside() {
		t.Fatal("an INACTIVE reviewer row must not grant approval scope")
	}

	if _, err := tx.Exec(`UPDATE `+entityid.ProductPlanStaff+` SET active = true, role = 'primary' WHERE id = 'live-test-reviewer-edge'`); err != nil {
		t.Fatalf("re-role: %v", err)
	}
	if !outside() {
		t.Fatal("a non-reviewer (primary) product_plan_staff row must not grant approval scope")
	}
}
