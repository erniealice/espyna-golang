//go:build postgresql

package core

// Shape tests for the column-less tenant register (columnLessTenantTables) and
// its single-hop parent-JOIN probes (columnLessTenantParentJoins).
//
// Why these exist: membership in these two maps is the ONLY thing that makes the
// decorator emit AUTHZ_WS_SHADOW_PASS on a column-less List and run the by-id
// ownership probe on Read/Update/Delete/HardDelete. A table that is simply absent
// gets silence — no predicate, no log — which is exactly how inventory_item stayed
// an unscoped cross-tenant surface through the 2026-05-30 hardening pass. A probe
// with a typo'd alias fails the same way in practice: deriveWorkspaceViaParent
// errors, and in SHADOW (the default) a probe error is deliberately swallowed and
// passed through, so nothing ever surfaces the mistake at runtime.

import (
	"strings"
	"testing"
)

// TestColumnLessTenantParentJoinsAreInternallyConsistent validates every probe in
// the register, not just the newest one: a probe whose fragments disagree
// assembles into SQL that can only error, and that error is invisible in SHADOW.
//
// The assembled query (deriveWorkspaceViaParent) is:
//
//	SELECT <parentWs> FROM <table> <childAlias>
//	  LEFT JOIN <parentTable> <parentAlias> ON <joinCond>
//	 WHERE <childAlias>.id = $1
//
// so each probe must satisfy: joinCond mentions the childAlias-qualified fkColumn,
// joinCond mentions the parentAlias, and parentWs is parentAlias-qualified.
func TestColumnLessTenantParentJoinsAreInternallyConsistent(t *testing.T) {
	for table, probe := range columnLessTenantParentJoins {
		t.Run(table, func(t *testing.T) {
			if probe.fkColumn == "" || probe.childAlias == "" ||
				probe.parentAlias == "" || probe.parentTable == "" ||
				probe.joinCond == "" || probe.parentWs == "" {
				t.Fatalf("probe for %q has an empty field: %+v", table, probe)
			}

			if probe.childAlias == probe.parentAlias {
				t.Errorf("probe for %q reuses alias %q for both child and parent — the assembled SQL would be ambiguous",
					table, probe.childAlias)
			}

			childFK := probe.childAlias + "." + probe.fkColumn
			if !strings.Contains(probe.joinCond, childFK) {
				t.Errorf("probe for %q: joinCond %q does not anchor on %q — the LEFT JOIN would not use the declared FK",
					table, probe.joinCond, childFK)
			}

			if !strings.Contains(probe.joinCond, probe.parentAlias+".") {
				t.Errorf("probe for %q: joinCond %q never references parent alias %q",
					table, probe.joinCond, probe.parentAlias)
			}

			if !strings.HasPrefix(probe.parentWs, probe.parentAlias+".") {
				t.Errorf("probe for %q: parentWs %q is not qualified by parent alias %q — it would resolve against the child table",
					table, probe.parentWs, probe.parentAlias)
			}
		})
	}
}

// TestColumnLessTenantParentJoinsAreRegisteredTables guards the ordering trap: a
// probe is only ever consulted from scopeColumnLessByParent, which is only reached
// when columnLessTenantTables[table] is true. A probe for an unregistered table is
// dead configuration that reads as protection.
func TestColumnLessTenantParentJoinsAreRegisteredTables(t *testing.T) {
	for table := range columnLessTenantParentJoins {
		if !columnLessTenantTables[table] {
			t.Errorf("table %q has a parent-JOIN probe but is absent from columnLessTenantTables — the probe is unreachable",
				table)
		}
	}
}

// TestInventoryItemRegisteredAsColumnLessTenant pins the W0 interim mitigation from
// docs/plan/20260729-inventory-item-tenant-scope (rider R1).
//
// ⚠ THIS TEST IS EXPECTED TO FAIL AT W4 — that is its job. Once the additive
// workspace_id column lands on BOTH education1 and professional1,
// tableHasWorkspaceColumn flips true, the direct-column path takes over and BOTH
// register entries must be removed (workspace_operations.go maintenance note).
// Delete this test in the same change that removes them; do not "fix" it by
// re-adding the entries.
func TestInventoryItemRegisteredAsColumnLessTenant(t *testing.T) {
	if !columnLessTenantTables["inventory_item"] {
		t.Fatal("inventory_item missing from columnLessTenantTables — the column-less List shadow log and the by-id probe are both gated on this entry")
	}

	probe, ok := columnLessTenantParentJoins["inventory_item"]
	if !ok {
		t.Fatal("inventory_item missing from columnLessTenantParentJoins — without a probe scopeColumnLessByParent degrades to shadow-PASS and can never deny by id")
	}

	// product is the only reliable anchor: location reaches a workspace for 0 of
	// the 13 live rows and product_variant has no workspace_id at all (plan §1.2).
	if probe.fkColumn != "product_id" {
		t.Errorf("inventory_item probe anchors on %q, want %q", probe.fkColumn, "product_id")
	}
	if probe.parentTable != "product" {
		t.Errorf("inventory_item probe joins %q, want %q", probe.parentTable, "product")
	}
	if probe.parentWs != probe.parentAlias+".workspace_id" {
		t.Errorf("inventory_item probe derives %q, want %q", probe.parentWs, probe.parentAlias+".workspace_id")
	}
}
