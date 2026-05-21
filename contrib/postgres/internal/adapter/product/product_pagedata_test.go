//go:build postgresql

package product

import "testing"

// TestGetProductItemPageData_ParityWithReadProduct asserts that
// GetProductItemPageData(id).GetProduct() is proto.Equal to
// ReadProduct(id).GetData()[0]. This is the canonical-pagedata invariant
// established in plan 20260429-pagedata-canonicalize.
//
// Implementation note: the PG adapter test harness in this directory does not
// yet have a shared fixture/seed setup; standing one up is out of scope for
// the canonicalization wave. Once a harness exists, this test should:
//  1. Insert a fully-populated product row (all proto fields).
//  2. Call GetProductItemPageData and ReadProduct.
//  3. Assert proto.Equal on the returned Product against the Read result.
//  4. Repeat with an inactive product and assert page-data returns "not found".
func TestGetProductItemPageData_ParityWithReadProduct(t *testing.T) {
	t.Skip("TODO: parity test — needs PG fixture harness")
}

// TestGetProductListPageData_ParityWithListProducts asserts that the products
// returned by GetProductListPageData are a (filter-active) subset of those
// returned by ListProducts, with the same proto field set per row.
func TestGetProductListPageData_ParityWithListProducts(t *testing.T) {
	t.Skip("TODO: parity test — needs PG fixture harness")
}
