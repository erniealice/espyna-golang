package schema

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	optionsv1 "github.com/erniealice/esqyma/pkg/schema/v1/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// forceImportZeroImporterTables are the resolved table names of the 5 pb packages
// that have ZERO importers in espyna today (their adapters are unwired). Their
// init() only runs because barrel.go blank-imports them; without that they would
// be silently absent from protoregistry.GlobalTypes. Build() asserts every one of
// these is present after the walk so a future regression (dropping a barrel import,
// or a rename) fails LOUD rather than silently shrinking the registry.
//
// See docs/plan/20260530-reflectionless-crud/phase0-findings.md §c (GAP-C).
var forceImportZeroImporterTables = []string{
	"asset_component",
	"asset_disposal",
	"asset_location",
	"asset_maintenance",
	"integration_config",
}

// minExpectedTables is a conservative lower bound on the number of table-annotated
// messages a FULLY-LINKED binary (one that imports the adapter barrel, i.e. the
// deployed server / the postgresql-tagged validator path) must discover. It
// backstops a catastrophic regression (an adapter-import refactor that drops most
// pb init()s, leaving GlobalTypes nearly empty) without being brittle against
// ongoing proto annotation work, which only ever ADDS tables. Phase 0 measured
// ~189 annotated messages; 201 proto files now carry table=true. 150 leaves
// generous head/tail room while still catching a collapse to a handful of tables.
//
// This floor is NOT enforced inside build()/Build() itself: in an isolated unit-test
// binary only the messages transitively imported by that test (plus the 5 barrel
// packages) register in protoregistry.GlobalTypes — legitimately far below 150.
// GlobalTypes population is incidental to the import graph (phase0-findings §c), so
// the floor is meaningful only once the adapter barrel is linked. The container
// wirePoint enforces it via AssertMinimumCoverage() after the adapter init()s have
// run; build() always enforces only the always-valid per-table force-import check.
const minExpectedTables = 150

var buildOnce sync.Once

// Build populates the Global registry from protoregistry.GlobalTypes (Q-DD2-A).
// It is idempotent (guarded by sync.Once) so it is safe to call from the
// container's init path and again from the validator's Global access.
//
// For each registered message it reads the (options.v1.table) MessageOptions
// extension; if table == true it resolves the table name (table_name override, else
// message-name -> snake_case), classifies the columns (Q-DD1=C), and stores them.
//
// After the walk it asserts the 5 zero-importer force-imported tables are present
// and the total clears the minimum, returning an error if either check fails. No SQL.
func Build() error {
	var buildErr error
	buildOnce.Do(func() {
		buildErr = build(Global)
	})
	return buildErr
}

// build performs one walk into the supplied registry. Separated from Build for
// testability (tests build into a fresh Registry without the sync.Once latch).
func build(reg *Registry) error {
	var walkErr error

	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		md := mt.Descriptor()

		table, ok := tableOptions(md)
		if !ok {
			return true // not a table-annotated message; skip.
		}

		reg.put(table, Classify(md))
		return true
	})

	if walkErr != nil {
		return walkErr
	}

	// Always-valid regression guard: the 5 force-imported zero-importer tables
	// register in ANY binary that imports this package (via barrel.go), so a miss
	// here is a real regression regardless of how much of the adapter graph is
	// linked. The total-count floor is deliberately NOT enforced here (see
	// minExpectedTables doc + AssertMinimumCoverage).
	return assertForceImports(reg)
}

// AssertMinimumCoverage enforces the total table-count floor. It is meaningful
// only in a fully-linked binary (the deployed server / the postgresql validator
// path) where every wired adapter's transitive imports have populated
// protoregistry.GlobalTypes. The container wirePoint calls this AFTER Build() so a
// catastrophic collapse of the import graph fails the boot loud. In a degraded
// (but non-collapsed) binary it logs and returns nil rather than blocking startup.
func AssertMinimumCoverage() error {
	if n := Global.Len(); n < minExpectedTables {
		return fmt.Errorf(
			"schema: discovered only %d table-annotated messages (expected >= %d) — protoregistry.GlobalTypes is under-populated; check adapter import graph",
			n, minExpectedTables,
		)
	}
	return nil
}

// tableOptions reads the (options.v1.table) MessageOptions extension and returns
// the resolved table name when table == true. Resolution: table_name override if
// non-empty, else the message name lowered to snake_case.
func tableOptions(md protoreflect.MessageDescriptor) (string, bool) {
	opts := md.Options()
	if opts == nil {
		return "", false
	}
	ext := proto.GetExtension(opts, optionsv1.E_Table)
	tableOpts, ok := ext.(*optionsv1.MessageOptions)
	if !ok || tableOpts == nil || !tableOpts.GetTable() {
		return "", false
	}
	if override := tableOpts.GetTableName(); override != "" {
		return override, true
	}
	return messageNameToSnake(string(md.Name())), true
}

// messageNameToSnake converts a PascalCase proto message name to its snake_case
// table name (AssetComponent -> asset_component, RevenueRunAttempt ->
// revenue_run_attempt). The table-annotated message names in this tree carry no
// acronym runs (no ID/URL/HTTP), so a simple boundary-insert is exact.
func messageNameToSnake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// assertForceImports is the Q-DD2 boot-shot prerequisite ("assert the barrel
// worked"). It fails loud if any force-imported zero-importer table is missing,
// which means barrel.go's blank import was dropped or the pb package was renamed.
// Because barrel.go is part of this package, these 5 tables register in every
// binary that imports schema, so this check is valid independent of the rest of
// the adapter import graph.
func assertForceImports(reg *Registry) error {
	var missing []string
	for _, t := range forceImportZeroImporterTables {
		if _, ok := reg.ColsFor(t); !ok {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf(
			"schema.Build: %d force-imported zero-importer table(s) absent from protoregistry.GlobalTypes (%s) — barrel.go blank import missing or pb package renamed",
			len(missing), strings.Join(missing, ", "),
		)
	}
	return nil
}
