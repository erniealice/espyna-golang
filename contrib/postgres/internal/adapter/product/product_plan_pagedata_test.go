//go:build postgresql

package product

import "testing"

// TestGetProductPlanItemPageData_ParityWithReadProductPlan asserts that
// GetProductPlanItemPageData(id).GetProductPlan() is proto.Equal to
// ReadProductPlan(id).GetData()[0]. Canonical-pagedata invariant per plan
// 20260429-pagedata-canonicalize.
//
// Implementation note: shared PG fixture harness is not yet in place for this
// adapter dir. Once available, this test should:
//   1. Insert a fully-populated product_plan row.
//   2. Call GetProductPlanItemPageData and ReadProductPlan.
//   3. Assert proto.Equal between the page-data ProductPlan and Read result.
//   4. Confirm inactive rows yield "not found" from page-data.
func TestGetProductPlanItemPageData_ParityWithReadProductPlan(t *testing.T) {
	t.Skip("TODO: parity test — needs PG fixture harness")
}

// TestGetProductPlanListPageData_ParityWithListProductPlans asserts that the
// product plans returned by GetProductPlanListPageData are a (filter-active)
// subset of those from ListProductPlans, with identical proto fields per row.
func TestGetProductPlanListPageData_ParityWithListProductPlans(t *testing.T) {
	t.Skip("TODO: parity test — needs PG fixture harness")
}
