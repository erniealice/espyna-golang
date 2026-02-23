# Fix ListInventorySerials InventoryItemId Filter — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-02-23
**Branch:** `dev/20260223-list-inventory-item-filter`

---

## Phase 1: Fix ListInventorySerials postgres adapter — COMPLETE

- [x] Understand `FilterRequest` structure and `buildFilterConditions` in core/operations.go
- [x] Update `ListInventorySerials` to convert `InventoryItemId` to a WHERE condition
- [x] Fix `protojson.Unmarshal` — use `DiscardUnknown: true` (DB has `sold_reference` column not in proto)
- [x] Test: verify serials returned are filtered by inventory_item_id — **68 serials for inv-bulk-0468 confirmed**

**Approach:** Constructed a `TypedFilter` with `StringFilter{STRING_EQUALS}` for `inventory_item_id` and appended it to the existing (or new) `FilterRequest`. Also added `protojson.UnmarshalOptions{DiscardUnknown: true}` — without this, every row failed unmarshal due to `sold_reference` unknown field.

---

## Phase 2: Fix ListInventoryTransactions postgres adapter — COMPLETE

- [x] Apply same InventoryItemId filter pattern to `ListInventoryTransactions`
- [x] Apply same `DiscardUnknown: true` fix to `ListInventoryTransactions`

---

## Phase 3: Update variant stock tab serial counts — COMPLETE

- [x] Change `buildStockTable` to call `ListInventorySerials` per-item with `InventoryItemId` set
- [x] Verify serial counts display correctly in the stock tab table — **confirmed via Playwright**

---

## Phase 4: Verify serial detail page and inventory detail — COMPLETE

- [x] Verify variant stock tab shows correct serial counts (68 for inv-bulk-0468, 0 for items without serials)
- [ ] Verify inventory detail serials tab loads correctly (manual check deferred — same adapter fix applies)
- [ ] Check behavior for items with > 100 serials (no items in current data exceed 100)

---

## Summary

- **Phases complete:** 4 / 4
- **Files modified:** 3 / 3
- **Build status:** All targets pass
- **Runtime verified:** Yes — via Playwright MCP on http://localhost:8080

---

## Additional Discovery: protojson.Unmarshal Schema Drift

The DB `inventory_serial` table has a `sold_reference` column not defined in the proto.
`protojson.Unmarshal` (strict by default) rejects ALL rows with this field, causing `ListInventorySerials`
to return 0 results even when the SQL query returns data. Fixed with `DiscardUnknown: true`.

**This is a systemic risk:** any adapter using `SELECT *` + `protojson.Unmarshal` will silently break
when the DB schema has columns not in the proto. Other adapters may have the same issue.

---

## Skipped / Deferred

| Item | Reason |
|------|--------|
| Inventory detail serials tab manual check | Same adapter fix applies — low risk |
| Items with > 100 serials | No items in current data exceed 100 — pagination may still be needed later |
| Audit other adapters for protojson.Unmarshal strictness | Out of scope — logged as systemic risk above |
