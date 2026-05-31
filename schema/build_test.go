package schema

import (
	"testing"

	treasuryv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// TestMessageNameToSnake covers the PascalCase message name -> snake_case table
// name conversion used when no table_name override is present.
func TestMessageNameToSnake(t *testing.T) {
	cases := map[string]string{
		"AssetComponent":           "asset_component",
		"IntegrationConfig":        "integration_config",
		"RevenueRunAttempt":        "revenue_run_attempt",
		"AccruedExpenseSettlement": "accrued_expense_settlement",
		"Job":                      "job",
		"EventRecurrence":          "event_recurrence",
	}
	for in, want := range cases {
		if got := messageNameToSnake(in); got != want {
			t.Errorf("messageNameToSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestTableOptionsResolvesOverride proves the table_name override is honored:
// message Collection (treasury/collection) resolves to "treasury_collection", not
// the message-name-derived "collection". This is the only override mechanism in
// the tree and must work from day one.
func TestTableOptionsResolvesOverride(t *testing.T) {
	md := (&treasuryv1.Collection{}).ProtoReflect().Descriptor()
	table, ok := tableOptions(md)
	if !ok {
		t.Fatalf("Collection must be table-annotated")
	}
	if table != "treasury_collection" {
		t.Fatalf("table_name override not honored: got %q, want %q (message-name-derived would be %q)",
			table, "treasury_collection", "collection")
	}
}

// TestBuildCoversZeroImporters runs a full walk into a fresh registry and confirms
// the barrel did its job: the zero-importer tables are present, the override
// table is present, and the coverage assertion passes. This exercises the real
// protoregistry.GlobalTypes (populated by every blank/transitive import in the
// test binary, including barrel.go).
func TestBuildCoversZeroImporters(t *testing.T) {
	reg := NewRegistry()
	if err := build(reg); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	for _, want := range forceImportZeroImporterTables {
		if _, ok := reg.ColsFor(want); !ok {
			t.Errorf("zero-importer table %q absent after build — barrel force-import failed", want)
		}
	}

	if _, ok := reg.ColsFor("treasury_collection"); !ok {
		t.Errorf("override table treasury_collection absent after build")
	}

	// NOTE: the total-count floor (minExpectedTables) is intentionally NOT asserted
	// here. In this isolated schema-package test binary only the messages
	// transitively imported by the tests (plus the barrel packages) register in
	// protoregistry.GlobalTypes — legitimately far below 150. The floor is enforced
	// by AssertMinimumCoverage() at the container wirePoint, where the full adapter
	// barrel is linked. See build.go minExpectedTables doc.
	t.Logf("registry populated with %d tables in the isolated test binary", reg.Len())
}

// TestGlobalBuildIdempotent confirms the package-level Build() is callable,
// idempotent, and populates the Global singleton without error.
func TestGlobalBuildIdempotent(t *testing.T) {
	if err := Build(); err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	n1 := Global.Len()
	if err := Build(); err != nil {
		t.Fatalf("second Build() error: %v", err)
	}
	if Global.Len() != n1 {
		t.Errorf("Build() not idempotent: %d then %d", n1, Global.Len())
	}

	// Spot-check the lookups consumed by operations.go feed the right kinds.
	if c, ok := Global.ColByName("job", "date_created"); !ok || !c.IsBigintMillis {
		t.Errorf("Global.ColByName(job, date_created) = %+v ok=%v, want IsBigintMillis", c, ok)
	}
}
