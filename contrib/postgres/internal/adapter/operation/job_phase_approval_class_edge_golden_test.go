//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// classEdgeOwnedSQLGolden is the byte-exact text classEdgeOwnedSQL emitted at
// base b388d68f (before the shared classEdgeFromSQL builder refactor), for the
// two placeholder layouts the shape suite uses. AC-O12.
const classEdgeOwnedSQLGolden43 = `EXISTS (
			SELECT 1
			FROM subscription_group_member m
			JOIN subscription_group sg
			       ON sg.id = m.subscription_group_id AND sg.status = 'current'
			JOIN subscription_group_product_plan c
			       ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = $3
			JOIN product_plan pp
			       ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id
			JOIN subscription_group_product_plan_staff e
			       ON e.subscription_group_product_plan_id = c.id AND e.active
			      AND e.staff_id = $4 AND e.workspace_id = $3
			      AND (e.job_template_phase_id IS NULL OR EXISTS (
			             SELECT 1
			             FROM job_template_phase ep, job_template_phase jpp
			             WHERE ep.id = e.job_template_phase_id
			               AND jpp.id = jp.template_phase_id
			               AND ep.phase_order = jpp.phase_order
			           ))
			WHERE m.client_id = j.client_id AND m.active AND m.workspace_id = $3
		)`

const classEdgeOwnedSQLGolden79 = `EXISTS (
			SELECT 1
			FROM subscription_group_member m
			JOIN subscription_group sg
			       ON sg.id = m.subscription_group_id AND sg.status = 'current'
			JOIN subscription_group_product_plan c
			       ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = $9
			JOIN product_plan pp
			       ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id
			JOIN subscription_group_product_plan_staff e
			       ON e.subscription_group_product_plan_id = c.id AND e.active
			      AND e.staff_id = $7 AND e.workspace_id = $9
			      AND (e.job_template_phase_id IS NULL OR EXISTS (
			             SELECT 1
			             FROM job_template_phase ep, job_template_phase jpp
			             WHERE ep.id = e.job_template_phase_id
			               AND jpp.id = jp.template_phase_id
			               AND ep.phase_order = jpp.phase_order
			           ))
			WHERE m.client_id = j.client_id AND m.active AND m.workspace_id = $9
		)`

func TestClassEdgeOwnedSQL_ByteIdenticalAfterBuilderRefactor(t *testing.T) {
	for _, c := range []struct {
		facet, ws int
		want      string
	}{{4, 3, classEdgeOwnedSQLGolden43}, {7, 9, classEdgeOwnedSQLGolden79}} {
		if got := classEdgeOwnedSQL(c.facet, c.ws); got != c.want {
			t.Errorf("classEdgeOwnedSQL(%d,%d) drifted from base:\n got: %q\nwant: %q", c.facet, c.ws, got, c.want)
		}
	}
}

// TestClassPrimaryEdgeOwnedSQLShape (design-v2 rule 4): the main-teacher
// fragment is the shared class-edge walk plus exactly the primary-role term.
func TestClassPrimaryEdgeOwnedSQLShape(t *testing.T) {
	got := classPrimaryEdgeOwnedSQL(5, 2)
	for _, frag := range []string{
		"e.active",
		"e.staff_id = $5 AND e.role = 'primary' AND e.workspace_id = $2",
		"m.client_id = j.client_id",
		"m.active AND m.workspace_id = $2",
		"sg.status = 'current'",
		"c.active AND c.workspace_id = $2",
		"pp.product_id = j.output_product_id",
		"e.job_template_phase_id IS NULL",
		"ep.phase_order = jpp.phase_order",
		"FROM " + entityid.SubscriptionGroupMember + " m",
	} {
		if !strings.Contains(got, frag) {
			t.Errorf("primary-edge fragment missing %q:\n%s", frag, got)
		}
	}
	// Same text as the any-role fragment apart from the primary term.
	any := classEdgeOwnedSQL(5, 2)
	if strings.Replace(got, " AND e.role = 'primary'", "", 1) != any {
		t.Errorf("primary-edge fragment must equal classEdgeOwnedSQL plus the role term only:\n%s\nvs\n%s", got, any)
	}
	if strings.Count(got, "'primary'") != 1 || strings.Contains(any, "primary") {
		t.Errorf("role term must appear exactly once, and only in the primary variant")
	}
	for _, stale := range []string{"$3", "$4"} {
		if strings.Contains(got, stale) {
			t.Errorf("stale placeholder %q", stale)
		}
	}
}
