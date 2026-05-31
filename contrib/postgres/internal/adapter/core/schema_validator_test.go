//go:build postgresql

package core

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/schema"
)

// buildRegistry ensures schema.Global is populated for the reconcile tests. In
// this postgresql test binary the adapter imports are NOT all linked, so only the
// tables transitively imported here (plus the 5 barrel packages) register — that is
// fine for the reconcile branch tests, which target specific known tables.
func buildRegistry(t *testing.T) {
	t.Helper()
	if err := schema.Build(); err != nil {
		t.Fatalf("schema.Build(): %v", err)
	}
}

// liveFromRegistry builds a synthetic "live" column map that exactly matches the
// descriptor for the given tables — i.e. a perfectly-in-sync database.
func liveFromRegistry(t *testing.T, tables ...string) map[string]map[string]bool {
	t.Helper()
	out := make(map[string]map[string]bool)
	for _, tbl := range tables {
		cols, ok := schema.ColsFor(tbl)
		if !ok {
			t.Fatalf("registry has no table %q (force-import/barrel issue?)", tbl)
		}
		m := make(map[string]bool, len(cols))
		for _, c := range cols {
			m[c.Name] = true
		}
		out[tbl] = m
	}
	return out
}

// TestReconcileCleanWhenInSync: a live schema that matches the descriptors and
// carries only allowlisted extra tables produces zero drift.
func TestReconcileCleanWhenInSync(t *testing.T) {
	buildRegistry(t)
	live := liveFromRegistry(t, "asset_component", "integration_config")
	// Add an allowlisted-only table (no descriptor) — must NOT be drift.
	live["payment_method"] = map[string]bool{"id": true, "name": true}

	drift, _ := reconcile(live, descriptorOutOfScope)
	if len(drift) != 0 {
		t.Fatalf("expected no drift, got:\n%s", strings.Join(drift, "\n"))
	}
}

// TestReconcileUnknownLiveTableIsDrift: a live table that is neither in the
// registry nor in the allowlist fails fast (Q-DD5=A strict boot-shot).
func TestReconcileUnknownLiveTableIsDrift(t *testing.T) {
	buildRegistry(t)
	live := liveFromRegistry(t, "asset_component")
	live["totally_unknown_table"] = map[string]bool{"id": true}

	drift, _ := reconcile(live, descriptorOutOfScope)
	if len(drift) == 0 {
		t.Fatal("expected drift for unknown live table, got none")
	}
	joined := strings.Join(drift, "\n")
	if !strings.Contains(joined, "totally_unknown_table") {
		t.Fatalf("drift did not name the unknown table: %s", joined)
	}
	if !strings.Contains(joined, "descriptorOutOfScope allowlist") {
		t.Fatalf("drift message should point at the allowlist: %s", joined)
	}
}

// TestReconcileAllowlistSuppressesDrift: the same unknown table, when allowlisted,
// produces no drift — proving the allowlist path (payment_method / revenue_payment
// et al.) works.
func TestReconcileAllowlistSuppressesDrift(t *testing.T) {
	buildRegistry(t)
	live := liveFromRegistry(t, "asset_component")
	live["revenue_payment"] = map[string]bool{"id": true, "amount": true}

	drift, _ := reconcile(live, descriptorOutOfScope)
	if len(drift) != 0 {
		t.Fatalf("allowlisted table revenue_payment must not be drift, got:\n%s", strings.Join(drift, "\n"))
	}
}

// TestReconcileMissingDescriptorColumnIsDrift: a descriptor column absent from the
// live table fails fast (branch 3).
func TestReconcileMissingDescriptorColumnIsDrift(t *testing.T) {
	buildRegistry(t)
	live := liveFromRegistry(t, "asset_component")
	// Drop one real column from the live map to simulate a DB missing it.
	var dropped string
	for col := range live["asset_component"] {
		dropped = col
		break
	}
	delete(live["asset_component"], dropped)

	drift, _ := reconcile(live, descriptorOutOfScope)
	if len(drift) == 0 {
		t.Fatalf("expected drift for descriptor column %q absent from live table", dropped)
	}
	if !strings.Contains(strings.Join(drift, "\n"), dropped) {
		t.Fatalf("drift should name the missing column %q: %v", dropped, drift)
	}
}

// TestReconcileDroppedRegistryTableWarnsNotErrors: a registry table with no live
// table (GAP-A: deferred_revenue / prepayment / security_deposit) must SKIP/WARN,
// never error. We simulate by registering a known table in the registry and then
// providing a live map that omits it.
func TestReconcileDroppedRegistryTableWarnsNotErrors(t *testing.T) {
	buildRegistry(t)
	// asset_component IS in the registry; omit it from live -> warning, not drift.
	live := map[string]map[string]bool{
		// only an allowlisted extra, no descriptor tables present live
		"payment_method": {"id": true},
	}

	drift, warnings := reconcile(live, descriptorOutOfScope)
	if len(drift) != 0 {
		t.Fatalf("a registry table missing from live must be a warning, not drift; got drift:\n%s",
			strings.Join(drift, "\n"))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warnings for registry tables absent from the live schema")
	}
	// asset_component (force-imported, always in registry) must be among the warnings.
	if !strings.Contains(strings.Join(warnings, "\n"), "asset_component") {
		t.Fatalf("expected a skip-warning for asset_component, got: %v", warnings)
	}
}

// TestAllowlistContainsDesignNamedEntries guards the design-named allowlist seeds.
func TestAllowlistContainsDesignNamedEntries(t *testing.T) {
	for _, want := range []string{"payment_method", "revenue_payment"} {
		if !descriptorOutOfScope[want] {
			t.Errorf("descriptorOutOfScope must contain design-named entry %q", want)
		}
	}
}
