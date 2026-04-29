//go:build postgresql

package subscription

import "testing"

// TestGetPlanItemPageData_FieldParityWithReadPlan asserts that the canonical
// page-data path returns a Plan proto field-equal to ReadPlan(id).GetData()[0].
// Today the GetPlanItemPageData body delegates to ReadPlan + an adjacent
// plan_locations lookup, so parity holds for every column on `plan` —
// drift-proof for new proto fields (no SELECT whitelist to maintain).
//
// TODO: enable once the package gains a Postgres test harness (insert one
// plan row + N plan_location rows, then proto.Equal-compare GetPlan() vs
// ReadPlan().GetData()[0] minus the explicit PlanLocations denorm).
func TestGetPlanItemPageData_FieldParityWithReadPlan(t *testing.T) {
	t.Skip("TODO: requires Postgres test harness (see file-level comment)")
}

// TestGetPlanListPageData_FieldParityWithListPlans asserts the same parity
// for the list path — every Plan in the page-data list has the same field
// values as the corresponding entry in ListPlans, plus the plan_locations
// denorm wired in.
//
// TODO: enable alongside the item parity test above.
func TestGetPlanListPageData_FieldParityWithListPlans(t *testing.T) {
	t.Skip("TODO: requires Postgres test harness (see file-level comment)")
}
