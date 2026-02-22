# Fix ListInventorySerials InventoryItemId Filter — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-02-23
**Branch:** `dev/20260223-list-inventory-item-filter`

---

## Phase 1: Fix ListInventorySerials postgres adapter — COMPLETE

- [x] Understand `FilterRequest` structure and `buildFilterConditions` in core/operations.go
- [x] Update `ListInventorySerials` to convert `InventoryItemId` to a WHERE condition
- [ ] Test: verify serials returned are filtered by inventory_item_id

**Approach:** Constructed a `TypedFilter` with `StringFilter{STRING_EQUALS}` for `inventory_item_id` and appended it to the existing (or new) `FilterRequest`. This reuses `buildFilterConditions` in core/operations.go — no raw SQL needed.

---

## Phase 2: Fix ListInventoryTransactions postgres adapter — COMPLETE

- [x] Apply same pattern to `ListInventoryTransactions`
- [ ] Test: verify transactions returned are filtered by inventory_item_id

**Approach:** Identical pattern to Phase 1.

---

## Phase 3: Update variant stock tab serial counts — COMPLETE

- [x] Change `buildStockTable` to call `ListInventorySerials` per-item with `InventoryItemId` set
- [ ] Verify serial counts display correctly in the stock tab table

**Approach:** Pre-filter variant items, then loop over each item calling `ListInventorySerials` with `InventoryItemId: &iid`. Each call returns only serials for that item (typically < 100), avoiding the 100-row limit issue.

---

## Phase 4: Verify serial detail page and inventory detail — NOT STARTED

- [ ] Verify variant serial detail page loads serials for the item
- [ ] Verify inventory detail serials tab loads correctly
- [ ] Check behavior for items with > 100 serials

---

## Summary

- **Phases complete:** 3 / 4 (Phase 4 is runtime verification only)
- **Files modified:** 3 / 3
- **Build status:** All three targets pass (`espyna`, `centymo`, `retail-admin`)

---

## Skipped / Deferred (update as you work)

| Item | Reason |
|------|--------|
| Runtime verification (Phase 4) | Requires running server with seeded DB — verify manually |

---

## How to Resume

To continue this work:
1. Read this progress file and the [plan](./plan.md)
2. Start the retail-admin server: `cd apps/retail-admin && powershell -ExecutionPolicy Bypass -File scripts/run.ps1`
3. Navigate to a product variant's Stock tab — serial counts should be non-zero for items with serials
4. Navigate to an inventory item's Serials tab — should show only serials for that item
5. Test a variant serial detail page — should show filtered serials
6. If any item has > 100 serials, check that the count matches `quantity_on_hand` (if not, the 100-row limit still applies and pagination or limit increase is needed)
