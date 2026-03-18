# Centralize Table Constants — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-03-16
**Branch:** `dev/20260316-centralize-table-constants`

---

## Phase 1: Add entity groups to entityid — COMPLETE

- [x] Add domain-level slices (14 slices: CommonEntities, EntityEntities, etc.)
- [x] Add consolidated `All` slice (101 entity IDs)
- [x] Verify: `go build ./registry/entityid/`

---

## Phase 2: Introduce `TableConfig` type — COMPLETE

- [x] Add `TableConfig` struct with `TableName()` method
- [x] Add `NewTableConfig()` and `NewDefaultTableConfig()` constructors
- [x] Add re-exports in `registry/registry.go`
- [x] Verify: `go build ./...`

---

## Phase 3: Migrate domain providers — COMPLETE

- [x] entity.go — signature + callsites (24 callsites, tryCreate simplified to 1-arg)
- [x] product.go — signature + callsites (16 calls)
- [x] subscription.go — signature + callsites (12 calls)
- [x] common.go — signature + callsites (2 calls)
- [x] event.go — signature + callsites (4 calls)
- [x] workflow.go — signature + callsites (6 calls)
- [x] revenue.go — signature + callsites (4 calls)
- [x] inventory.go — signature + callsites (6 calls)
- [x] expenditure.go — signature + callsites (4 calls)
- [x] integration.go — signature + callsites (1 call)
- [x] ledger.go — signature + callsites (2 calls)
- [x] treasury.go — signature + callsites (2 calls)
- [x] operation.go — signature + callsites (14 calls)
- [x] registry.go (domain) — field type updated
- [x] registry.go (providers) — parameter type updated

---

## Phase 4: Migrate provider manager and container — COMPLETE

- [x] Update `providers/manager.go` type + accessor
- [x] Update `core/container.go` type references + `.Client` → `.TableName("client")`
- [x] Update `core/usecases.go` — no changes needed (types flow through)
- [x] Update `TableConfigBuilder` type in database.go
- [x] Update `BuildDatabaseTableConfig` return type

---

## Phase 5: Migrate provider builders — COMPLETE

- [x] Rewrite PostgreSQL `buildPgTableConfig()` — 107 lines → 10-line `entityid.All` loop
- [x] Rewrite Firestore `buildTableConfig()` — same map-based pattern
- [x] Update Mock builder — `DefaultDatabaseTableConfig()` → `NewDefaultTableConfig()`
- [x] Fix `cmd/seeder/main.go` — 6 field accesses → `TableName()` calls

---

## Phase 6: Remove old code — COMPLETE

- [x] Delete `DatabaseTableConfig` struct from registry/database.go
- [x] Delete `DefaultDatabaseTableConfig()` function
- [x] Delete `internal/composition/config/application.go` (stale duplicate)
- [x] Remove dead `WithDatabaseTableConfig`, `createTableConfig`, `DatabaseTableConfigSetter`
- [x] Clean up `registry/registry.go` re-exports

---

## Phase 7: Verification — COMPLETE

- [x] `go build ./...` passes (espyna + postgres + google contribs)
- [x] `go vet ./...` passes
- [x] No remaining `DatabaseTableConfig` type references (grep confirms zero)
- [x] retail-admin builds successfully
- [x] service-admin builds successfully
- [x] retail-client builds successfully
- [x] 74 `TableName()` calls across 13 domain provider files

---

## Summary

- **Phases complete:** 7 / 7
- **Files modified:** ~25 (13 domain providers + 6 infrastructure + 3 adapters + seeder + registry re-exports)
- **Files deleted:** 1 (config/application.go)
- **Lines removed:** ~300+ (old struct, defaults, stale duplicate, dead options code)
- **Lines added:** ~80 (TableConfig type, entityid slices, simplified builders)

---

## Skipped / Deferred

| Item | Reason |
|------|--------|
| Rename `BuildDatabaseTableConfig` → `BuildTableConfig` | Function name still says "Database" but works with `*TableConfig`. Cosmetic — defer to avoid unnecessary churn |
