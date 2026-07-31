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

// TestInventoryItemGraduatedFromColumnLessRegister is the W4 counterpart of the W0
// entry-exists assertion it replaces (docs/plan/20260729-inventory-item-tenant-scope).
//
// inventory_item now has a real workspace_id column on both databases, so
// tableHasWorkspaceColumn returns TRUE and the decorator takes the DIRECT-column
// path: injectWorkspaceFilter on List, the by-id ownership check on
// Read/Update/Delete/HardDelete. Re-adding either register entry would be an active
// regression, not a belt-and-braces addition — membership routes the table down the
// column-less branch instead, and that branch cannot filter a List at all (a
// StringFilter predicate cannot express a parent JOIN), so the whole-list leak the
// column closed would silently reopen.
//
// The direct-column behaviour this graduation depends on is pinned separately in
// inventory_item_workspace_shape_test.go; this test guards only the register.
func TestInventoryItemGraduatedFromColumnLessRegister(t *testing.T) {
	if columnLessTenantTables["inventory_item"] {
		t.Error("inventory_item is back in columnLessTenantTables — it has a direct workspace_id column, and the column-less branch leaves List entirely unfiltered")
	}

	if _, ok := columnLessTenantParentJoins["inventory_item"]; ok {
		t.Error("inventory_item is back in columnLessTenantParentJoins — the product_id parent-JOIN probe is superseded by the direct workspace_id column")
	}
}
