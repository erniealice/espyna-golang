# Revenue PostgreSQL Factory + Checkout Patch — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-02-23
**Branch:** `dev/20260223-revenue-postgres-factory`
**Last saved:** 2026-02-23 (Phase 5 debugging)

---

## Phase 1: Revenue + Line Item Factories — DONE

- [x] Create `postgres/revenue/revenue.go` — CRUD + List + GetListPageData + GetItemPageData
- [x] Create `postgres/revenue_line_item/revenue_line_item.go` — CRUD + List + GetListPageData + GetItemPageData
- [x] Verify `go build -tags postgresql` passes for espyna

---

## Phase 2: Supporting Factories — DONE

- [x] Create `postgres/revenue_category/revenue_category.go` — CRUD + List + ListPageData + ItemPageData
- [x] Create `postgres/revenue_attribute/revenue_attribute.go` — CRUD + List + ListPageData + ItemPageData

---

## Phase 3: Centymo Double-Division Fix — DONE

- [x] Fix `service.go:153` — removed `/100.0` from Amount (Maya adapter already converts)

---

## Phase 4: Retail-Client Patch — DONE

- [x] Wire centymo CheckoutService in `container.go` (replace raw SQL + PaymentIntegration)
- [x] Replace PaymentIntegration struct with ProcessWebhookFunc in `handler.go`
- [x] Simplify `checkout_post.go` (removed manual Maya session block + unused imports)
- [x] Update `views.go` (replaced PaymentUC with ProcessWebhook)
- [x] Implement UpdateOrderStatus in `order_checkout.go` (was no-op, now calls UpdateRevenue UC)
- [x] Added UpdateRevenueFunc + ptr helper to `order_checkout.go`

---

## Phase 5: Verification — IN PROGRESS (3 bugs found and fixed)

- [x] Build passes for espyna (`go build -tags postgresql`)
- [x] Build passes for retail-client (`go build -tags "gcp_storage,google,mock_auth,maya,noop,postgresql,vanilla"`)
- [x] Server starts with centymo active: "Orders: centymo CheckoutService (via espyna UCs)"
- [ ] Manual test: checkout → Maya → confirmation
- [ ] E2E tests pass (218 tests)

### Bug 1: Factory imports missing — FIXED
- **Symptom:** Server log `Orders: raw SQL order service (revenue UCs not available)` — centymo not activated
- **Root cause:** New factory packages not blank-imported in `postgres/imports.go` — `init()` never ran
- **Fix:** Added 4 blank imports to `packages/espyna-golang-ryta/.../postgres/imports.go`:
  - `_ ".../postgres/revenue"`
  - `_ ".../postgres/revenue_attribute"`
  - `_ ".../postgres/revenue_category"`
  - `_ ".../postgres/revenue_line_item"`

### Bug 2: Authorization failed for guest checkout — FIXED
- **Symptom:** `checkout: create revenue: Authorization failed`
- **Root cause:** espyna `authcheck.Check()` calls `RequireUserIDFromContext(ctx)` — guest checkout has no user in context. Mock auth is enabled+allowAll, but the check fails BEFORE calling mock auth (line 34 of authcheck.go).
- **Fix:** Added `consumer.WithUserID(ctx, systemUserID)` in `order_checkout.go` before calling centymo. Uses `"system-checkout"` as service user ID. Applied to PlaceOrder, GetOrder, and UpdateOrderStatus.

### Bug 3: date/time field value out of range — FIXED
- **Symptom:** `pq: date/time field value out of range` on revenue INSERT
- **Root cause:** Proto `RevenueDate` is int64 millis (e.g. `1771886746000`). Factory serializes proto→JSON→map via protojson, which outputs millis as a JSON string. `dbOps.Create` inserts this raw string into a postgres `timestamp` column → "out of range".
- **Fix:** Added `convertMillisToTime()` helper in revenue factory. Called after JSON unmarshal, before dbOps.Create/Update, for `revenueDate`, `dateCreated`, `dateModified` fields. Converts millis > 1e12 to `time.Time`.

### Bug 4: create revenue returns empty response — INVESTIGATING
- **Symptom:** `checkout: create revenue: unexpected empty response`
- **Root cause:** TBD — the INSERT may succeed but `dbOps.Create` returns data that doesn't unmarshal back to proto correctly, OR the revenue table is missing columns that the factory tries to INSERT (checkout_session_id, payment_provider, fulfillment_type, delivery_address are NOT in the DB yet).
- **Likely fix:** The `dbOps.Create` probably ignores unknown columns, but the response mapping may fail. Need to check if the INSERT actually succeeded in the DB, and whether the response proto unmarshal is losing the data.

---

## Summary

- **Phases complete:** 4 / 5 (Phase 5 debugging in progress)
- **Files created:** 4 new factory files + 1 adapter (order_checkout.go)
- **Files modified:** 9 total (see list below)
- **Bugs found in Phase 5:** 4 (3 fixed, 1 investigating)

---

## Skipped / Deferred

| Item | Reason |
|------|--------|
| — | — |

---

## How to Resume

To continue this work:
1. Read this progress file and the [plan](./plan.md)
2. Server is running as background task `bbaa2d4`
3. **Resume debugging Bug 4:** "create revenue: unexpected empty response"

### Debugging Bug 4

The error comes from centymo `service.go:99-100`:
```go
if !createRevenueResp.GetSuccess() || len(createRevenueResp.GetData()) == 0 {
    return nil, fmt.Errorf("checkout: create revenue: unexpected empty response")
}
```

Possible causes:
1. `dbOps.Create` returns nil/empty because the INSERT fails silently (missing columns?)
2. The response proto doesn't include `Success: true` (factory bug — check if `CreateRevenueResponse` sets Success)
3. The proto unmarshal fails because returned DB columns don't match proto field names

**Action items:**
- Check if the revenue factory's `CreateRevenue` sets `Success: true` on the response (it does NOT — line 90-92 only returns Data, no Success field)
- Check if `dbOps.Create` actually inserted the row (query DB: `SELECT * FROM revenue ORDER BY date_created DESC LIMIT 1`)
- Check if the proto→JSON round-trip loses data due to column mismatch

**Key insight about Bug 4:** Looking at the factory code (line 90-92), `CreateRevenueResponse` is returned with `Data` but **no `Success: true`**. The centymo service checks `GetSuccess()` — this will return `false` by default! Compare with how other factories handle it (e.g. product_option).

### DB info
- Database: `mono2` (NOT `ryta_retail`)
- revenue table: 15 columns (no checkout_session_id/payment_provider/fulfillment_type/delivery_address yet)
- revenue_line_item: 14 columns (no price_list_id/variant_id/variant_label/location_id/cost_price yet)
- revenue_category: 8 columns
- revenue_attribute: 7 columns

### Files created across all sessions
1. `packages/espyna-golang-ryta/.../postgres/revenue/revenue.go` (NEW — Phase 1)
2. `packages/espyna-golang-ryta/.../postgres/revenue_line_item/revenue_line_item.go` (NEW — Phase 1)
3. `packages/espyna-golang-ryta/.../postgres/revenue_category/revenue_category.go` (NEW — Phase 2)
4. `packages/espyna-golang-ryta/.../postgres/revenue_attribute/revenue_attribute.go` (NEW — Phase 2)
5. `apps/retail-client/internal/domain/order_checkout.go` (NEW — Phase 4 prep)

### Files modified across all sessions
1. `packages/centymo-golang-ryta/checkout/service.go` — Phase 3 (double-division fix)
2. `apps/retail-client/internal/composition/container.go` — Phase 4 (centymo wiring)
3. `apps/retail-client/internal/composition/views.go` — Phase 4 (ProcessWebhook replaces PaymentUC)
4. `apps/retail-client/internal/presentation/checkout/handler.go` — Phase 4 (ProcessWebhookFunc replaces PaymentIntegration)
5. `apps/retail-client/internal/presentation/checkout/checkout_post.go` — Phase 4 (removed manual Maya session)
6. `apps/retail-client/internal/presentation/checkout/webhook.go` — Phase 4 (uses h.processWebhook)
7. `apps/retail-client/internal/domain/order_checkout.go` — Phase 4 (UpdateOrderStatus + auth context)
8. `packages/espyna-golang-ryta/.../postgres/imports.go` — Phase 5 Bug 1 (added 4 revenue factory imports)
9. `packages/espyna-golang-ryta/.../postgres/revenue/revenue.go` — Phase 5 Bug 3 (convertMillisToTime)
