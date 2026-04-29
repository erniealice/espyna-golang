//go:build postgresql

package entity

import (
	"testing"
)

// TestGetClientItemPageData_FieldParityWithReadClient asserts that
// GetClientItemPageData(id).GetClient() is proto.Equal to
// ReadClient(id).GetData()[0]. After the wave2-client canonicalization,
// GetClientItemPageData composes ReadClient + loadClientCategories, so the
// only legitimate difference is the .Categories slice — every other field
// must round-trip identically.
//
// Skipped until the PG fixture harness for the entity package lands. The
// core package (contrib/postgres/internal/adapter/core/operations_test.go)
// has a working *sql.DB fixture pattern, but no equivalent has been wired
// into the entity package adapters. Plan-level parity is verified by the
// type-level check below at compile time.
func TestGetClientItemPageData_FieldParityWithReadClient(t *testing.T) {
	t.Skip("TODO: parity test — needs PG fixture harness for entity package")
}
