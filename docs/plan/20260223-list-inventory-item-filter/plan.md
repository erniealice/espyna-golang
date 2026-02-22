# Fix ListInventorySerials InventoryItemId Filter — Design Plan

**Date:** 2026-02-23
**Branch:** `dev/20260223-list-inventory-item-filter`
**Status:** Draft
**App/Package:** espyna-golang-ryta (primary), centymo-golang-ryta (consumer)

---

## Overview

The `ListInventorySerials` postgres adapter ignores the `InventoryItemId` field on `ListInventorySerialsRequest`, and the generic `dbOps.List()` caps results at 100 rows. This causes the variant stock tab to show "Serial Count: 0" for all items — it only gets 100 of 8,367 serials, none of which match the current variant's inventory items. The same issue affects `ListInventoryTransactions` and likely other list operations with entity-specific filter fields.

---

## Motivation

### Problem Statement

The espyna generic `List()` at `packages/espyna-golang-ryta/.../postgres/core/operations.go:387` has:
```go
limit := int32(100) // Default limit
```

And the `ListInventorySerials` adapter at `packages/espyna-golang-ryta/.../postgres/inventory_serial/inventory_serial.go:190` only passes `req.Filters` to the generic list — it **ignores** the `InventoryItemId` proto field:
```go
var params *interfaces.ListParams
if req != nil && req.Filters != nil {
    params = &interfaces.ListParams{Filters: req.Filters}
}
listResult, err := r.dbOps.List(ctx, r.tableName, params)
```

This means:
1. Even when callers set `InventoryItemId` (e.g., `inventory/detail/page.go:422`), the filter is silently ignored
2. The response is capped at 100 rows regardless
3. The `buildStockTable` function (`variant/page.go:351`) calls it with an empty request expecting ALL serials — gets only 100

### Impact

| Call site | File | Problem |
|-----------|------|---------|
| Variant stock tab serial counts | `centymo/.../variant/page.go:351` | Shows 0 for all items (100/8367 returned) |
| Inventory detail serials tab | `centymo/.../inventory/detail/page.go:422` | Sets `InventoryItemId` but filter ignored — gets wrong 100 serials |
| Variant serial detail page | `centymo/.../variant/serial/page.go:207` | Sets `InventoryItemId` but filter ignored |
| Inventory dashboard | `centymo/.../inventory/dashboard/page.go:213` | Counts only first 100 serials |

### Root Cause

The `ListInventorySerials` (and `ListInventoryTransactions`) postgres adapters were auto-generated from a template that only maps the generic `Filters` field to `dbOps.List()`. Entity-specific filter fields defined in the proto (`InventoryItemId`) are never converted into SQL WHERE conditions.

---

## Architecture

### Current flow (broken)

```
View code sets InventoryItemId on request
  → ListInventorySerials use case (passes through)
    → Postgres adapter (IGNORES InventoryItemId, uses only Filters)
      → dbOps.List() with LIMIT 100
        → Returns wrong 100 rows
```

### Target flow (fixed)

```
View code sets InventoryItemId on request
  → ListInventorySerials use case (passes through)
    → Postgres adapter converts InventoryItemId to FilterRequest condition
      → dbOps.List() with inventory_item_id = ? AND LIMIT 100
        → Returns correct filtered rows (typically < 100 per item)
```

### Alternative considered: Direct SQL in centymo

We could add a raw SQL `COUNT` query in the centymo view layer. This would bypass the espyna adapter entirely. However, this:
- Breaks the layered architecture (views shouldn't know about SQL)
- Only fixes the count case, not the serial listing case
- Doesn't fix `ListInventoryTransactions` which has the same issue

**Decision:** Fix the adapter layer — it's the correct place for this, and fixes all consumers at once.

---

## Implementation Steps

### Phase 1: Fix ListInventorySerials postgres adapter

The adapter must convert `InventoryItemId` into a filter condition before calling `dbOps.List()`.

- **Step 1:** In `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/inventory_serial/inventory_serial.go:190`, update `ListInventorySerials` to check `req.InventoryItemId` and build a `FilterRequest` with a condition `inventory_item_id = ?`
- **Step 2:** Check how `interfaces.ListParams.Filters` works — specifically how `FilterRequest` is structured (`packages/esqyma-ryta/pkg/schema/v1/domain/common/common.pb.go`) to add a field equality filter
- **Step 3:** If `FilterRequest` doesn't support simple field equality cleanly, add `inventory_item_id` as a direct SQL WHERE clause. The adapter already has access to `r.db` for direct queries (see `GetInventorySerialListPageData` at line 226)

### Phase 2: Fix ListInventoryTransactions postgres adapter (same pattern)

- Apply the same fix to `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/inventory_transaction/inventory_transaction.go:189`
- The `ListInventoryTransactionsRequest` also has an `InventoryItemId` field that is ignored

### Phase 3: Update variant stock tab to use filtered counts

After Phase 1 fixes the adapter, the `buildStockTable` function in `centymo/.../variant/page.go:348-360` can be optimized:
- Instead of loading ALL serials (even filtered, this could be large), call `ListInventorySerials` once per inventory item with `InventoryItemId` set
- Or better: keep the current bulk approach but it now returns filtered data per the adapter fix

Actually, the bulk approach in `buildStockTable` calls with an **empty** request (no `InventoryItemId`). Even with the adapter fix, it still hits the 100-row limit for the unfiltered case. The proper fix for this specific call site is:
- **Option A:** Loop over each inventory item and call `ListInventorySerials` with `InventoryItemId` set (N queries, but each returns < 100 rows)
- **Option B:** Add a new direct SQL method to count serials grouped by inventory_item_id for a set of item IDs

**Recommended: Option A** — it's simple, works within the existing architecture, and N is small (typically 10-30 items per variant).

### Phase 4: Verify serial detail page works

- The new `variant/serial/page.go:207` already sets `InventoryItemId` — after Phase 1, it will return correct filtered serials
- But verify it still works with items that have > 100 serials (some items have `quantity_on_hand` up to 100)

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang-ryta/.../postgres/inventory_serial/inventory_serial.go` | Honor `InventoryItemId` in `ListInventorySerials` | 1 |
| `packages/esqyma-ryta/pkg/schema/v1/domain/common/common.pb.go` | Read-only — understand `FilterRequest` structure | 1 |
| `packages/espyna-golang-ryta/.../postgres/core/operations.go` | Read-only — understand `buildFilterConditions` | 1 |
| `packages/espyna-golang-ryta/.../postgres/inventory_transaction/inventory_transaction.go` | Honor `InventoryItemId` in `ListInventoryTransactions` | 2 |
| `packages/centymo-golang-ryta/views/product/detail/variant/page.go` | Update `buildStockTable` to count serials per-item | 3 |
| `packages/centymo-golang-ryta/views/product/detail/variant/serial/page.go` | Verify — should work after Phase 1 | 4 |
| `packages/centymo-golang-ryta/views/inventory/detail/page.go` | Verify — `loadSerials` should work after Phase 1 | 4 |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** 10
**Estimated files to modify:** 3
**Estimated context usage:** Low (< 30 files)

No sub-agents needed. Single session is sufficient.

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| `dbOps.List` limit of 100 still applies for items with > 100 serials | Medium — some serial lists truncated | Phase 4 verification; if needed, raise limit for filtered queries or add pagination to serial table |
| Changing adapter behavior may affect other callers | Low — adding filter is additive, unfiltered calls behave the same | Review all callers of `ListInventorySerials` (4 call sites, all in centymo views) |

**Dependencies:**
- Phase 2 is independent from Phase 1 (same pattern, different file)
- Phase 3 depends on Phase 1 (needs working filter)
- Phase 4 depends on Phase 1 (verification only)

---

## Acceptance Criteria

- [ ] `ListInventorySerials` with `InventoryItemId` set returns only serials for that item
- [ ] `ListInventoryTransactions` with `InventoryItemId` set returns only transactions for that item
- [ ] Variant stock tab shows correct serial counts (non-zero for items with serials)
- [ ] Variant serial detail page shows all serials for the selected inventory item
- [ ] Inventory detail serials tab shows correct serials for the item
- [ ] Build passes with `go build -tags "google_uuidv7,mock_auth,mock_storage,noop,postgresql,vanilla"`

---

## Design Decisions

**Why fix the adapter, not the view?** The `InventoryItemId` field exists on the proto request specifically for this purpose. The adapter simply never implemented it. Fixing at the adapter layer is the architecturally correct solution — all consumers benefit, and views don't need to know about SQL.

**Why per-item queries (Option A) for stock tab counts?** A grouped COUNT query would be more efficient but requires either a new method on the adapter or raw SQL in the view. Since N (inventory items per variant) is small (10-30), N queries is acceptable and keeps the code simple. Can be optimized later if needed.
