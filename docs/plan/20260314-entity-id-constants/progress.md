# Entity ID Constants — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-03-14
**Branch:** `dev/20260314-entity-id-constants`

---

## Phase 1: Create `registry/entityid` package — NOT STARTED

- [ ] Create `packages/espyna-golang-ryta/registry/entityid/entityid.go`
- [ ] Define all 51 entity key constants grouped by domain
- [ ] Verify build: `go build ./registry/entityid/`

---

## Phase 2: Migrate PostgreSQL adapters — NOT STARTED

- [ ] Migrate `contrib/postgres/internal/adapter/common/` (1 file)
- [ ] Migrate `contrib/postgres/internal/adapter/attribute_value/` (1 file)
- [ ] Migrate `contrib/postgres/internal/adapter/entity/` (17 files)
- [ ] Migrate `contrib/postgres/internal/adapter/event/` (2 files)
- [ ] Migrate `contrib/postgres/internal/adapter/product/` (11 files)
- [ ] Migrate `contrib/postgres/internal/adapter/product_option*/` (3 files)
- [ ] Migrate `contrib/postgres/internal/adapter/product_variant*/` (3 files)
- [ ] Migrate `contrib/postgres/internal/adapter/revenue*/` (4 files)
- [ ] Migrate `contrib/postgres/internal/adapter/expenditure/` (4 files)
- [ ] Migrate `contrib/postgres/internal/adapter/inventory_*/` (6 files)
- [ ] Migrate `contrib/postgres/internal/adapter/subscription/` (12 files)
- [ ] Migrate `contrib/postgres/internal/adapter/treasury/` (2 files)
- [ ] Migrate `contrib/postgres/internal/adapter/ledger/` (1 file)
- [ ] Migrate `contrib/postgres/internal/adapter/document/` (2 files)
- [ ] Migrate `contrib/postgres/internal/adapter/integrations/` (1 file)

---

## Phase 3: Migrate Firestore adapters — NOT STARTED

- [ ] Migrate all 53 Firestore adapter files in `contrib/google/internal/database/firestore/`

---

## Phase 4: Migrate Mock adapters — NOT STARTED

- [ ] Migrate all 43 Mock adapter files in `internal/infrastructure/adapters/secondary/database/mock/`

---

## Phase 5: Migrate consumer sites — NOT STARTED

- [ ] Migrate `internal/composition/providers/domain/entity.go`
- [ ] Migrate `internal/composition/providers/domain/product.go`
- [ ] Migrate `internal/composition/providers/domain/subscription.go`
- [ ] Migrate `internal/composition/providers/domain/common.go`
- [ ] Migrate `internal/composition/providers/domain/event.go`
- [ ] Migrate `internal/composition/providers/domain/workflow.go`
- [ ] Migrate `internal/composition/providers/domain/revenue.go`
- [ ] Migrate `internal/composition/providers/domain/inventory.go`
- [ ] Migrate `internal/composition/providers/domain/expenditure.go`
- [ ] Migrate `internal/composition/providers/domain/integration.go`
- [ ] Migrate `internal/composition/providers/domain/ledger.go`
- [ ] Migrate `internal/composition/providers/domain/treasury.go`

---

## Phase 6: Verification — NOT STARTED

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] No remaining hardcoded entity strings in RegisterRepositoryFactory calls
- [ ] No remaining hardcoded entity strings in CreateRepository calls

---

## Summary

- **Phases complete:** 0 / 6
- **Files modified:** 0 / 178

---

## Skipped / Deferred (update as you work)

| Item | Reason |
|------|--------|
| — | — |

---

## How to Resume

To continue this work:
1. Read this progress file and the [plan](./plan.md)
2. Check git status for any uncommitted changes from the previous session
3. Start from the first incomplete phase above
4. Phase 1 must be done first (creates the constants file)
5. Phases 2-5 can run in parallel via sub-agents after Phase 1
6. Phase 6 (verification) runs last
7. Update checkboxes and summary as you complete steps

**Key files to read first:**
- `packages/espyna-golang-ryta/registry/entityid/entityid.go` (if Phase 1 done)
- `packages/espyna-golang-ryta/internal/infrastructure/registry/database.go:84-131` (DatabaseTableConfig for reference)
- Sample adapter: `packages/espyna-golang-ryta/contrib/postgres/internal/adapter/entity/client.go`
- Sample consumer: `packages/espyna-golang-ryta/internal/composition/providers/domain/entity.go`
