# Revenue PostgreSQL Factory + Checkout Patch — Design Plan

**Date:** 2026-02-23
**Branch:** `dev/20260223-revenue-postgres-factory`
**Status:** Draft
**App/Package:** espyna (primary), centymo (bugfix), retail-client (patch)

---

## Overview

Add PostgreSQL repository factories for the revenue domain (revenue, revenue_line_item, revenue_category, revenue_attribute) to espyna. These factories are the missing link that prevented centymo's CheckoutService from working — the composition provider (`revenue.go`) already exists and tries to create repositories via `registry.CreateRepository("revenue", ...)`, but no `postgresql:revenue` factory was registered. Once the factories exist, patch retail-client to use centymo's CheckoutService instead of the current raw SQL workaround.

---

## Motivation

The current retail-client checkout uses two disconnected systems:
1. **Raw SQL** (`order_db.go`) for revenue/line-item creation
2. **espyna payment UCs** (via `PaymentIntegration` struct) for Maya session/webhook

This was a workaround because espyna had no `postgresql:revenue` factory. It means:
- Revenue creation logic is duplicated (raw SQL vs centymo)
- No stock reservation via espyna inventory UCs (raw SQL instead)
- No serial assignment
- No checkout session ID stored on revenue record
- `PaymentIntegration` is a fragile function-pointer wrapper

Adding the factories lets centymo's CheckoutService handle the full flow: revenue → line items → stock → serials → payment — all through espyna UCs.

---

## Architecture

### Current (workaround):
```
retail-client checkout_post.go
  → postgresOrderService (raw SQL: INSERT revenue, INSERT revenue_line_item, UPDATE inventory)
  → PaymentIntegration.CreateCheckoutSession (espyna UC function pointer)
```

### Target:
```
retail-client checkout_post.go
  → centymo CheckoutService.PlaceOrder
    → espyna Revenue UC → postgresql:revenue factory
    → espyna RevenueLineItem UC → postgresql:revenue_line_item factory
    → espyna InventoryItem UC → postgresql:inventory_item factory (already exists)
    → espyna InventorySerial UC → postgresql:inventory_serial factory (already exists)
    → espyna Payment UC → Maya adapter (already exists)
```

### Factory Pattern (from product_option reference):
```go
//go:build postgresql
package revenue

func init() {
    registry.RegisterRepositoryFactory("postgresql", "revenue", func(conn any, tableName string) (any, error) {
        db := conn.(*sql.DB)
        dbOps := postgresCore.NewPostgresOperations(db)
        return NewPostgresRevenueRepository(dbOps, tableName), nil
    })
}

type PostgresRevenueRepository struct {
    revenuepb.UnimplementedRevenueDomainServiceServer
    dbOps     interfaces.DatabaseOperation
    db        *sql.DB
    tableName string
}
// Implements: Create, Read, Update, Delete, List, GetListPageData, GetItemPageData
```

---

## Implementation Steps

### Phase 1: Revenue Factory (critical path)

1. **Create `revenue/revenue.go` factory** — implements `RevenueDomainServiceServer`
   - File: `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/revenue/revenue.go` **(NEW)**
   - Follow `product_option.go` pattern exactly
   - Methods: CreateRevenue, ReadRevenue, UpdateRevenue, DeleteRevenue, ListRevenues
   - GetRevenueListPageData CTE: JOIN client for client_name, JOIN location for location_name
   - GetRevenueItemPageData CTE: same JOINs, single row
   - Revenue proto fields: id, date_created, date_modified, active, name, client_id, revenue_date, revenue_date_string, total_amount, currency, status, reference_number, notes, revenue_category_id, location_id, checkout_session_id, payment_provider, fulfillment_type, delivery_address

2. **Create `revenue_line_item/revenue_line_item.go` factory** — implements `RevenueLineItemDomainServiceServer`
   - File: `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/revenue_line_item/revenue_line_item.go` **(NEW)**
   - Methods: Create, Read, Update, Delete, List, GetListPageData, GetItemPageData
   - RevenueLineItem proto fields: id, date_created, date_modified, active, revenue_id, product_id, description, quantity, unit_price, total_price, notes, line_item_type, inventory_item_id, inventory_serial_id, price_list_id, variant_id, variant_label, location_id, cost_price
   - GetListPageData CTE: JOIN revenue, JOIN product for enriched display

### Phase 2: Supporting Revenue Factories

3. **Create `revenue_category/revenue_category.go` factory** — implements `RevenueCategoryDomainServiceServer`
   - File: `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/revenue_category/revenue_category.go` **(NEW)**
   - Standard CRUD + List (no GetListPageData/GetItemPageData needed initially)

4. **Create `revenue_attribute/revenue_attribute.go` factory** — implements `RevenueAttributeDomainServiceServer`
   - File: `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/postgres/revenue_attribute/revenue_attribute.go` **(NEW)**
   - Standard CRUD + List

### Phase 3: Fix Centymo Double-Division Bug

5. **Fix amount conversion in centymo CheckoutService**
   - File: `packages/centymo-golang-ryta/checkout/service.go:153`
   - Current: `Amount: float64(req.TotalAmount) / 100.0` — WRONG (Maya adapter already divides by 100)
   - Fix: `Amount: float64(req.TotalAmount)` — pass centavos, let adapter convert
   - Same bug as the one just fixed in `checkout_post.go`

### Phase 4: Patch Retail-Client to Use Centymo

6. **Wire centymo CheckoutService in container.go**
   - File: `apps/retail-client/internal/composition/container.go:155-172`
   - Replace raw SQL order service with centymo's CheckoutService:
     ```go
     checkoutDeps := checkout.CheckoutDeps{
         CreateRevenue:    ucs.Revenue.Revenue.CreateRevenue.Execute,
         UpdateRevenue:    ucs.Revenue.Revenue.UpdateRevenue.Execute,
         ReadRevenue:      ucs.Revenue.Revenue.ReadRevenue.Execute,
         ListRevenues:     ucs.Revenue.Revenue.ListRevenues.Execute,
         CreateLineItem:   ucs.Revenue.RevenueLineItem.CreateRevenueLineItem.Execute,
         ListLineItems:    ucs.Revenue.RevenueLineItem.ListRevenueLineItems.Execute,
         // ... inventory + payment deps already available
     }
     checkoutSvc := checkout.NewService(checkoutDeps)
     orderService = domain.NewCheckoutOrderService(checkoutSvc, cartService)
     ```
   - Remove `PaymentIntegration` struct wiring

7. **Simplify checkout handler**
   - File: `apps/retail-client/internal/presentation/checkout/handler.go`
   - Remove `PaymentIntegration` struct (centymo handles payment internally)
   - Remove `paymentUC` field from Handler

8. **Simplify checkout_post.go**
   - File: `apps/retail-client/internal/presentation/checkout/checkout_post.go:94-134`
   - Remove the manual `CreateCheckoutSession` block — centymo's PlaceOrder does this internally
   - The result already contains `CheckoutURL` from centymo

9. **Update views.go**
   - File: `apps/retail-client/internal/composition/views.go:53`
   - Remove PaymentUC parameter from checkout.NewHandler

10. **Keep order_db.go as fallback** (optional)
    - Don't delete — it can serve as fallback when centymo is unavailable
    - Container can check if revenue UCs are available and fall back to raw SQL

### Phase 5: Verification

11. **Build verification** — `go build -tags "gcp_storage,google,mock_auth,maya,noop,postgresql,vanilla" ./...`
12. **Manual test** — checkout → Maya redirect → payment → confirmation
13. **Run E2E tests** — `cd apps/retail-client/tests && pnpm test`

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang-ryta/.../postgres/revenue/revenue.go` | **New file** — postgresql:revenue factory | 1 |
| `packages/espyna-golang-ryta/.../postgres/revenue_line_item/revenue_line_item.go` | **New file** — postgresql:revenue_line_item factory | 1 |
| `packages/espyna-golang-ryta/.../postgres/revenue_category/revenue_category.go` | **New file** — postgresql:revenue_category factory | 2 |
| `packages/espyna-golang-ryta/.../postgres/revenue_attribute/revenue_attribute.go` | **New file** — postgresql:revenue_attribute factory | 2 |
| `packages/centymo-golang-ryta/checkout/service.go` | Fix double /100 on Amount (line 153) | 3 |
| `apps/retail-client/internal/composition/container.go` | Wire centymo CheckoutService via espyna UCs | 4 |
| `apps/retail-client/internal/composition/views.go` | Remove PaymentUC param | 4 |
| `apps/retail-client/internal/presentation/checkout/handler.go` | Remove PaymentIntegration struct | 4 |
| `apps/retail-client/internal/presentation/checkout/checkout_post.go` | Remove manual Maya session creation | 4 |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** ~25 (proto schemas, reference factory, use cases, centymo service, retail-client composition)
**Estimated files to modify:** 5 existing + 4 new = 9
**Estimated context usage:** Medium (30-40 files with proto schemas)

**Sub-agent plan:**
- Phase 1-2: Each factory file is ~400 lines of mechanical code. Can be parallelized — revenue + revenue_line_item in one agent, revenue_category + revenue_attribute in another
- Phase 3: Single-line fix, no agent needed
- Phase 4: Sequential — container depends on Phase 1-2 being complete

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| Revenue table columns don't match proto fields | CTE queries fail | Verify with `\d revenue` in psql before writing CTE |
| Centymo CheckoutService assumes UCs return data in specific format | Order creation fails | Test with centymo's existing tests first |
| Inventory UC for serial assignment requires additional factories | Serial reservation fails | Keep best-effort (centymo already logs and continues) |

**Dependencies:**
- Phase 3 is independent (can be done immediately)
- Phase 4 depends on Phase 1 (revenue + revenue_line_item factories must exist)
- Phase 2 can run in parallel with Phase 4 (category + attribute not needed for checkout)

---

## Acceptance Criteria

- [ ] `go build` passes for espyna with `postgresql` build tag
- [ ] `go build` passes for retail-client with full tag set
- [ ] Server logs show "Revenue: revenue repositories initialized" (or similar)
- [ ] Checkout creates revenue record via centymo → espyna UCs (not raw SQL)
- [ ] Revenue line items created for each cart item
- [ ] Maya checkout session created with correct amount (centavos → pesos, no double division)
- [ ] Webhook updates revenue status via centymo
- [ ] Confirmation page shows correct order data
- [ ] Existing E2E tests pass (218 tests)
- [ ] retail-admin revenue list still works (factories serve both apps)

---

## Design Decisions

**Why not just fix raw SQL and keep the workaround?** The raw SQL in `order_db.go` duplicates centymo's checkout logic — revenue creation, line items, stock reservation — all without the benefits of espyna's UC layer (authorization, transactions, ID generation, audit). The factory approach unlocks centymo for both retail-client AND retail-admin, where revenue management is already wired through espyna UCs (they just fail at factory lookup).

**Why implement all 4 factories?** The composition provider (`revenue.go`) creates ALL 4 repositories in sequence. If any one factory is missing, the entire revenue domain fails to initialize. Even though checkout only needs revenue + revenue_line_item, the provider would error on revenue_category and revenue_attribute. We could add nil-checks to the provider, but implementing all 4 is cleaner and unblocks future admin features.

**Why keep order_db.go?** It serves as a zero-dependency fallback when centymo isn't available (e.g., development without espyna revenue UCs). Container can check if revenue repos initialized and fall back to raw SQL gracefully.
