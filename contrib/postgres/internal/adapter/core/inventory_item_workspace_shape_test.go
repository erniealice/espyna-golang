//go:build postgresql

package core

// R3 shape test — docs/plan/20260729-inventory-item-tenant-scope (Option B, rider R3).
//
// The HAZ-02 precedent (contrib/postgres/internal/adapter/operation/
// job_outcome_summary_shape_test.go) pins a workspace predicate by asserting on the
// STRING a hand-written SQL builder emits. inventory_item has no such builder: after
// Q-DEAD its two hand-written CTE methods are deleted, and every live path
// (/inventory/list/{location}, /inventory/detail/{id}, the dashboard's three
// ListInventoryItems calls, and the update/delete/hard-delete actions) runs through
// the generic decorator — ListInventoryItems -> r.dbOps.List, ReadInventoryItem ->
// r.dbOps.Read (contrib/postgres/internal/adapter/inventory/inventory_item.go:214,
// :116). So the analogous pin is on the DECORATOR's injected predicate, not on a SQL
// string: assert that WorkspaceAwareOperations prepends the workspace_id StringFilter
// on List and fail-closes on Read once tableHasWorkspaceColumn("inventory_item")
// returns true.
//
// WHY THIS PASSES WITHOUT THE LIVE COLUMN (no build tag, no skip guard needed):
// tableHasWorkspaceColumn consults a per-instance columnCache before it ever queries
// information_schema (workspace_operations.go:649-690). Seeding that cache is the
// supported way to simulate the post-migration world offline — it is exactly what
// newStubWorkspaceOps in workspace_operations_test.go already does for test_table.
// Every case below is hermetic: the hasColumn=true cases take the direct-column
// branch (no parent-JOIN probe, no *sql.DB use), and the hasColumn=false case
// exercises List only, which never touches the DB. Nothing here connects anywhere.
//
// WHAT THIS TEST CANNOT PROVE: that the column actually exists on education1 and
// professional1. The cache seed asserts the CONTRACT, not the schema. The DB is the
// arbiter for W3 — see the §6 post-change psql checks and the behavioural proof
// (foreign-workspace principal sees 0 list rows and a 404 on detail-by-id), both
// recorded in progress.md.
//
// Reuses stubInner / newCtxWithWorkspace from workspace_operations_test.go (same
// package, same //go:build postgresql tag).

import (
	"context"
	"database/sql"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/model"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// inventoryItemTable is read from the registry rather than hard-coded: the decorator
// keys BOTH the column cache and the column-less register on the table name the
// adapter passes down (PostgresInventoryItemRepository defaults tableName to
// "inventory_item", and adapter.go resolves it from registry/entityid
// unconditionally per Q-TABLE-NAMES). A drift between the constant and the real table
// name would silently disable every predicate below, so the test binds to the
// constant the production path uses.
var inventoryItemTable = entityid.InventoryItem

// capturingInner records the *ListParams the decorator hands to the inner
// DatabaseOperation. stubInner.List discards its params, so it cannot witness the
// injection; embedding it and overriding List keeps the other seven methods without
// editing the shared stub.
type capturingInner struct {
	*stubInner
	lastListTable  string
	lastListParams *interfaces.ListParams
}

func (c *capturingInner) List(_ context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	c.lastListTable = tableName
	c.lastListParams = params
	return &interfaces.ListResult{}, nil
}

// newInventoryItemOps builds a decorator whose column cache is pre-seeded for
// inventory_item, so tableHasWorkspaceColumn answers from memory and never queries
// information_schema. hasWorkspaceColumn=true simulates the post-W3 world (the
// additive migration applied); false is today's baseline.
//
// The *sql.DB is opened offline purely to populate the struct field, mirroring
// newStubWorkspaceOps; no test below reaches a code path that dereferences it.
func newInventoryItemOps(inner interfaces.DatabaseOperation, hasWorkspaceColumn bool) *WorkspaceAwareOperations {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")

	ops := &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
	cols := map[string]bool{"id": true, "product_id": true, "location_id": true, "sku": true}
	if hasWorkspaceColumn {
		cols["workspace_id"] = true
	}
	ops.columnCache[inventoryItemTable] = cols
	return ops
}

// locationFilter is a stand-in for the real request filter the mounted list page
// sends (/inventory/list/{location} filters by location_id). The workspace predicate
// must be added ALONGSIDE it, never instead of it.
func locationFilter(locationID string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: "location_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         locationID,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}
}

// ─── List: the predicate the whole plan exists to install ────────────────────

// TestInventoryItemListInjectsWorkspaceFilterOnceColumnExists is the core R3 pin.
// ListInventoryItems is the ONLY thing standing between a foreign principal and every
// tenant's inventory rows (the dead CTE list was never reachable — see the W4 deadness
// worksheet in progress.md), and it delegates verbatim to this decorator.
func TestInventoryItemListInjectsWorkspaceFilterOnceColumnExists(t *testing.T) {
	inner := &capturingInner{stubInner: &stubInner{}}
	w := newInventoryItemOps(inner, true)
	ctx := newCtxWithWorkspace("ws-owner")

	callerParams := &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{locationFilter("loc-1")},
		},
	}

	if _, err := w.List(ctx, inventoryItemTable, callerParams); err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if inner.lastListTable != inventoryItemTable {
		t.Fatalf("inner List received table %q, want %q", inner.lastListTable, inventoryItemTable)
	}
	got := inner.lastListParams
	if got == nil || got.Filters == nil || len(got.Filters.Filters) == 0 {
		t.Fatalf("inner List received no filters — inventory_item list is UNSCOPED (cross-workspace IDOR): %+v", got)
	}

	ws := got.Filters.Filters[0]
	if ws.GetField() != "workspace_id" {
		t.Fatalf("first filter is %q, want the injected workspace_id predicate (injectWorkspaceFilter PREPENDS): %+v",
			ws.GetField(), got.Filters.Filters)
	}
	sf := ws.GetStringFilter()
	if sf == nil {
		t.Fatal("workspace predicate is not a StringFilter — the query builder would not emit an equality predicate")
	}
	if sf.GetValue() != "ws-owner" {
		t.Errorf("workspace predicate binds %q, want the SESSION workspace %q (never a request param)", sf.GetValue(), "ws-owner")
	}
	if sf.GetOperator() != commonpb.StringOperator_STRING_EQUALS {
		t.Errorf("workspace predicate operator is %v, want STRING_EQUALS — a LIKE/contains match would leak sibling workspaces", sf.GetOperator())
	}
	if !sf.GetCaseSensitive() {
		t.Error("workspace predicate is case-insensitive; workspace ids are exact tokens and a fold could collide across tenants")
	}

	// The caller's own filter must survive: scoping is additive, not a replacement.
	if len(got.Filters.Filters) != 2 || got.Filters.Filters[1].GetField() != "location_id" {
		t.Errorf("caller's location_id filter was dropped by the injection: %+v", got.Filters.Filters)
	}

	// injectWorkspaceFilter clones; mutating the caller's ListParams would corrupt a
	// retried or shared request.
	if len(callerParams.Filters.Filters) != 1 || callerParams.Filters.Filters[0].GetField() != "location_id" {
		t.Errorf("caller's ListParams was mutated in place: %+v", callerParams.Filters.Filters)
	}
}

// TestInventoryItemListInjectsWorkspaceFilterWithNilParams covers the dashboard's
// unfiltered ListInventoryItems calls, which pass an empty ListParams. A nil-filter
// list is the widest possible read and must still be scoped.
func TestInventoryItemListInjectsWorkspaceFilterWithNilParams(t *testing.T) {
	inner := &capturingInner{stubInner: &stubInner{}}
	w := newInventoryItemOps(inner, true)
	ctx := newCtxWithWorkspace("ws-owner")

	if _, err := w.List(ctx, inventoryItemTable, nil); err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	got := inner.lastListParams
	if got == nil || got.Filters == nil || len(got.Filters.Filters) != 1 {
		t.Fatalf("nil-params list was not scoped — an unfiltered inventory read spans every workspace: %+v", got)
	}
	if got.Filters.Filters[0].GetField() != "workspace_id" ||
		got.Filters.Filters[0].GetStringFilter().GetValue() != "ws-owner" {
		t.Errorf("nil-params list carries the wrong predicate: %+v", got.Filters.Filters[0])
	}
}

// TestInventoryItemListUnscopedForIdentitylessContext pins the deliberate
// service-to-service / pre-auth carve-out (getWorkspaceID is a SOFT
// identity.FromContext read, never identity.Must — the login-500 footgun documented
// at workspace_operations.go:629-644). No identity means no predicate, and that is
// the contract, not a regression.
func TestInventoryItemListUnscopedForIdentitylessContext(t *testing.T) {
	inner := &capturingInner{stubInner: &stubInner{}}
	w := newInventoryItemOps(inner, true)

	if _, err := w.List(context.Background(), inventoryItemTable, nil); err != nil {
		t.Fatalf("List returned error (a panic here would be identity.Must creeping back in): %v", err)
	}
	if inner.lastListParams != nil {
		t.Errorf("identity-less context got a filter injected: %+v", inner.lastListParams)
	}
}

// TestInventoryItemListStaysUnfilteredWhileColumnMissing pins Option C's D-veto
// (plan §3): registering inventory_item in columnLessTenantTables — the W0 interim
// mitigation — buys a SHADOW LOG on List and a by-id probe, but List is
// shadow-only in BOTH modes because the StringFilter mechanism cannot express a
// parent-JOIN predicate (workspace_operations.go:421-431). The reported leak (the
// unscoped list) therefore stays open until the real column lands.
//
// This test exists so nobody mistakes W0 for the fix. It is window-safe: it holds
// whether inventory_item is currently in the register (W0 landed) or not (before W0 /
// after the W4 removal), because neither state filters List.
func TestInventoryItemListStaysUnfilteredWhileColumnMissing(t *testing.T) {
	inner := &capturingInner{stubInner: &stubInner{}}
	w := newInventoryItemOps(inner, false) // pre-migration: no workspace_id column
	ctx := newCtxWithWorkspace("ws-owner")

	if _, err := w.List(ctx, inventoryItemTable, nil); err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if inner.lastListParams != nil {
		t.Fatalf("column-less inventory_item list unexpectedly carried a filter (%+v) — if the register learned to rewrite List as a JOIN, update this test and plan §3 Option C",
			inner.lastListParams)
	}
}

// ─── Read: fail-closed by id ─────────────────────────────────────────────────

// assertNotFound asserts the decorator's fail-closed verdict: a 404 DatabaseError, so
// a cross-tenant id is indistinguishable from a nonexistent one (no oracle).
func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a fail-closed error, got nil — the row was returned across workspaces")
	}
	dbErr, ok := err.(*model.DatabaseError)
	if !ok {
		t.Fatalf("expected *model.DatabaseError, got %T (%v)", err, err)
	}
	if dbErr.HTTPStatus != 404 {
		t.Errorf("fail-closed status is %d, want 404 — a 403 would confirm the row exists in another tenant", dbErr.HTTPStatus)
	}
}

// TestInventoryItemReadFailsClosedOnForeignWorkspace is the by-id half of the plan:
// /inventory/detail/{id} must 404, not render another tenant's item.
func TestInventoryItemReadFailsClosedOnForeignWorkspace(t *testing.T) {
	inner := &stubInner{readResult: map[string]any{
		"id":           "item-1",
		"sku":          "SKU-1",
		"workspace_id": "ws-other",
	}}
	w := newInventoryItemOps(inner, true)

	result, err := w.Read(newCtxWithWorkspace("ws-owner"), inventoryItemTable, "item-1")
	assertNotFound(t, err)
	if result != nil {
		t.Errorf("rejected read still returned a row: %v", result)
	}
}

// TestInventoryItemReadFailsClosedOnNullWorkspace pins the backfill's orphan
// semantics. The W2 backfill deliberately leaves rows with a NULL/dangling product_id
// at workspace_id NULL (plan §5.1) — inventory_item declares NO foreign keys at all,
// so product_id is a bare nullable text and orphans are structurally possible. Those
// rows MUST be unreachable, not universally visible.
func TestInventoryItemReadFailsClosedOnNullWorkspace(t *testing.T) {
	inner := &stubInner{readResult: map[string]any{
		"id":           "item-orphan",
		"product_id":   nil,
		"workspace_id": nil, // backfill left it NULL: no derivable tenant anchor
	}}
	w := newInventoryItemOps(inner, true)

	_, err := w.Read(newCtxWithWorkspace("ws-owner"), inventoryItemTable, "item-orphan")
	assertNotFound(t, err)
}

// TestInventoryItemReadFailsClosedOnEmptyWorkspace covers the empty-string twin of the
// NULL case (a backfill that writes an empty string instead of leaving NULL must not
// become a wildcard).
func TestInventoryItemReadFailsClosedOnEmptyWorkspace(t *testing.T) {
	inner := &stubInner{readResult: map[string]any{
		"id":           "item-blank",
		"workspace_id": "",
	}}
	w := newInventoryItemOps(inner, true)

	_, err := w.Read(newCtxWithWorkspace("ws-owner"), inventoryItemTable, "item-blank")
	assertNotFound(t, err)
}

// TestInventoryItemReadAllowsOwnWorkspace is the false-positive guard: the fix must
// not break the 13 live professional1 rows, all of which backfill to
// default-workspace.
func TestInventoryItemReadAllowsOwnWorkspace(t *testing.T) {
	inner := &stubInner{readResult: map[string]any{
		"id":           "item-2",
		"workspace_id": "default-workspace",
	}}
	w := newInventoryItemOps(inner, true)

	result, err := w.Read(newCtxWithWorkspace("default-workspace"), inventoryItemTable, "item-2")
	if err != nil {
		t.Fatalf("in-tenant read was rejected: %v", err)
	}
	if result == nil || result["id"] != "item-2" {
		t.Fatalf("in-tenant read returned %v, want the row", result)
	}
}

// ─── Writes: the half the original finding did not name ──────────────────────

// TestInventoryItemWritesFailClosedOnForeignWorkspace pins that this was never a
// read-only disclosure. Update/Delete/HardDelete gate on the same ownership Read, so
// a cross-tenant id must not mutate or soft-delete another workspace's stock.
func TestInventoryItemWritesFailClosedOnForeignWorkspace(t *testing.T) {
	foreign := map[string]any{"id": "item-3", "workspace_id": "ws-other"}
	ctx := newCtxWithWorkspace("ws-owner")

	t.Run("update", func(t *testing.T) {
		w := newInventoryItemOps(&stubInner{readResult: foreign}, true)
		_, err := w.Update(ctx, inventoryItemTable, "item-3", map[string]any{"quantity_on_hand": 999})
		assertNotFound(t, err)
	})
	t.Run("delete", func(t *testing.T) {
		w := newInventoryItemOps(&stubInner{readResult: foreign}, true)
		assertNotFound(t, w.Delete(ctx, inventoryItemTable, "item-3"))
	})
	t.Run("harddelete", func(t *testing.T) {
		w := newInventoryItemOps(&stubInner{readResult: foreign}, true)
		assertNotFound(t, w.HardDelete(ctx, inventoryItemTable, "item-3"))
	})
}

// TestInventoryItemCreateStampsTenantAnchor closes the "new rows have no owner" hole
// (plan §3 Option B, D lens): before the column existed, Create injected nothing, so
// every row written was permanently unscoped. It also pins gate H1 for this table —
// a client-supplied camelCase workspaceId must not survive to collide with the
// trusted value during camel→snake normalization.
func TestInventoryItemCreateStampsTenantAnchor(t *testing.T) {
	w := newInventoryItemOps(&stubInner{}, true)
	ctx := newCtxWithWorkspace("ws-owner")

	out, err := w.Create(ctx, inventoryItemTable, map[string]any{
		"id":          "item-new",
		"sku":         "SKU-NEW",
		"workspaceId": "ws-attacker", // client-supplied camelCase spelling
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if out["workspace_id"] != "ws-owner" {
		t.Errorf("Create stamped workspace_id=%v, want the session workspace %q", out["workspace_id"], "ws-owner")
	}
	if _, ok := out["workspaceId"]; ok {
		t.Error("client camelCase workspaceId survived Create — tenant-reassignment collision vector open on inventory_item")
	}
}
