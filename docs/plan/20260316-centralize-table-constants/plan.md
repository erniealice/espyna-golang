# Centralize Table Constants — Design Plan

**Date:** 2026-03-16
**Branch:** `dev/20260316-centralize-table-constants`
**Status:** Draft
**Package:** espyna-golang

---

## Overview

Replace the `DatabaseTableConfig` struct (and its stale duplicate) with a map-based `TableConfig` that derives default table names directly from `entityid` constants. This eliminates quintuple duplication: adding a new entity currently requires updating 5 files; after this change, it requires only 1 (the entityid constant).

---

## Motivation

### Current state — 5 sources of truth for entity→table mappings

| # | File | What it maintains |
|---|------|-------------------|
| 1 | `registry/entityid/entityid.go` | `Client = "client"` — the constant |
| 2 | `internal/infrastructure/registry/database.go` | `Client string` — struct field + `Client: "client"` default |
| 3 | `internal/composition/config/application.go` | **Stale duplicate** struct + defaults (different Go type!) |
| 4 | `contrib/postgres/internal/adapter/adapter.go` | `Client: prefix + getPostgresTableEnv("CLIENT", "client")` |
| 5 | `internal/composition/options/infrastructure/database.go` | `Client: GetEnv(prefix+"CLIENT", "client")` (legacy path) |

### Consequences of the duplication

- **Drift:** 8 entityid constants have no struct field (JobActivity, ActivityLabor, ActivityMaterial, ActivityExpense, JobSettlement, InventoryMovement, License, LicenseHistory). ~15 struct fields have no entityid constant (Manager, WorkspaceClient, Session, etc.).
- **Two types:** `registry.DatabaseTableConfig` vs `config.DatabaseTableConfig` — different Go types with different field sets, used in different code paths.
- **Maintenance cost:** Every new entity requires touching 5 files with identical information.

### After this change — 1 source of truth

Add an entityid constant → it automatically works as a table name. Override via env vars if needed. Zero struct fields to maintain.

---

## Architecture

### Core insight

The entityid constant value IS the default table name:

```go
entityid.Client = "client"  →  default table name = "client"
```

There's no transformation. The `DatabaseTableConfig` struct exists only to allow per-provider overrides (e.g., `POSTGRES_TABLE_CLIENT=my_clients`). A map does this more naturally.

### New `TableConfig` type

```go
// TableConfig resolves entity names to table/collection names.
// Default table name = entityid constant value. Overrides stored in map.
type TableConfig struct {
    prefix    string            // optional prefix for all table names
    overrides map[string]string // entity → custom table name (only non-defaults)
}

// TableName returns the table/collection name for an entity.
func (tc *TableConfig) TableName(entity string) string {
    if tc == nil {
        return entity // safe nil — entityid IS the default
    }
    if override, ok := tc.overrides[entity]; ok {
        return tc.prefix + override
    }
    return tc.prefix + entity
}
```

### Simplified domain provider pattern

```go
// Before: two arguments, both carrying the same "client" value
tryCreate(entityid.Client, dbTableConfig.Client)

// After: single argument, table name derived automatically
tryCreate(entityid.Client)
// where tryCreate internally calls tableConfig.TableName(entity)
```

### Simplified provider builder pattern

```go
// Before: 100+ field assignments
func buildPgTableConfig() *registry.DatabaseTableConfig {
    return &registry.DatabaseTableConfig{
        Client: prefix + getPostgresTableEnv("CLIENT", "client"),
        // ... 100+ more
    }
}

// After: scan env vars, only store overrides
func buildPgTableConfig() *registry.TableConfig {
    prefix := getEnv("POSTGRES_TABLE_PREFIX", "")
    overrides := make(map[string]string)
    for _, entry := range entityid.All {
        envKey := "POSTGRES_TABLE_" + strings.ToUpper(strings.ReplaceAll(entry, " ", "_"))
        if val := os.Getenv(envKey); val != "" {
            overrides[entry] = val
        }
    }
    return registry.NewTableConfig(prefix, overrides)
}
```

### Dependency diagram

```
entityid/entityid.go  ← SINGLE source of truth (constants + All slice)
    ↑
registry/database.go  ← TableConfig type (map-based, no struct fields)
    ↑
contrib/postgres/     ← buildPgTableConfig() scans env vars → overrides map
    ↑
providers/domain/*.go ← tryCreate(entityid.X) → tableConfig.TableName(X)
```

---

## Implementation Steps

### Phase 1: Add entity groups to entityid package

Add domain-level slices and a consolidated `All` slice to `registry/entityid/entityid.go`:

```go
var CommonEntities = []string{Attribute, AttributeValue, Category}
var EntityEntities = []string{Admin, Client, ClientAttribute, ...}
// ... per domain
var All = buildAll() // concatenation of all domain slices
```

- File: `packages/espyna-golang/registry/entityid/entityid.go`
- Zero breaking changes — only additions
- Verify: `go build ./registry/entityid/`

### Phase 2: Introduce `TableConfig` type alongside old struct

Add the new `TableConfig` type to the registry without removing the old struct:

- Add `TableConfig` struct with `TableName()` method to `internal/infrastructure/registry/database.go`
- Add `NewTableConfig(prefix, overrides)` and `NewDefaultTableConfig()` constructors
- Add re-exports in `registry/registry.go`
- **Backward-compatible** — old struct still exists, nothing references new type yet
- Verify: `go build ./...`

### Phase 3: Migrate domain providers to `TableConfig`

Update all 14 domain provider files to accept `*registry.TableConfig` instead of `*registry.DatabaseTableConfig`:

- Change function signature: `dbTableConfig *registry.DatabaseTableConfig` → `tableConfig *registry.TableConfig`
- Simplify `tryCreate` helper to single-argument pattern:
  ```go
  tryCreate := func(entity string) interface{} {
      repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
  }
  ```
- Update all callsites from `tryCreate(entityid.X, dbTableConfig.X)` to `tryCreate(entityid.X)`
- For operation.go: same pattern, replace `repoCreator.CreateRepository(entityid.X, conn, dbTableConfig.X)` with `repoCreator.CreateRepository(entityid.X, conn, tableConfig.TableName(entityid.X))`

Files:
- `internal/composition/providers/domain/entity.go`
- `internal/composition/providers/domain/product.go`
- `internal/composition/providers/domain/subscription.go`
- `internal/composition/providers/domain/common.go`
- `internal/composition/providers/domain/event.go`
- `internal/composition/providers/domain/workflow.go`
- `internal/composition/providers/domain/revenue.go`
- `internal/composition/providers/domain/inventory.go`
- `internal/composition/providers/domain/expenditure.go`
- `internal/composition/providers/domain/integration.go`
- `internal/composition/providers/domain/ledger.go`
- `internal/composition/providers/domain/treasury.go`
- `internal/composition/providers/domain/operation.go`

### Phase 4: Migrate provider manager and container

- Update `providers/manager.go`: change `dbTableConfig *registry.DatabaseTableConfig` → `*registry.TableConfig`
- Update `GetDBTableConfig()` return type
- Update `core/container.go` and `core/usecases.go`: pass `*registry.TableConfig` to domain providers
- Update `BuildDatabaseTableConfig()` to return `*registry.TableConfig`

Files:
- `internal/composition/providers/manager.go:35,64-72,225-230`
- `internal/composition/core/container.go` (type references)
- `internal/composition/core/usecases.go` (all `GetDBTableConfig()` calls)

### Phase 5: Migrate provider-specific builders

Update PostgreSQL and other adapter builders to return `*registry.TableConfig`:

- **PostgreSQL:** Replace `buildPgTableConfig()` 100+ field assignments with env var scan loop using `entityid.All`
- **Firestore:** Same pattern if builder exists
- **Mock:** Same pattern if builder exists
- Update `RegisterDatabaseTableConfigBuilder` signature: `func() *DatabaseTableConfig` → `func() *TableConfig`
- Old `TableConfigBuilder` type alias updated

Files:
- `contrib/postgres/internal/adapter/adapter.go:34,39-146`
- `contrib/google/internal/database/firestore/` (if builder exists)
- `internal/infrastructure/adapters/secondary/database/mock/` (if builder exists)
- `internal/infrastructure/registry/database.go` (builder type)

### Phase 6: Remove old code and fix stale duplicate

- Delete `DatabaseTableConfig` struct from `internal/infrastructure/registry/database.go`
- Delete `DefaultDatabaseTableConfig()` function
- Delete entire `internal/composition/config/application.go` (stale duplicate type)
- Update or remove `internal/composition/options/infrastructure/database.go` `createTableConfig()` and `WithDatabaseTableConfig()` — must use new `TableConfig` type
- Remove old re-exports from `registry/registry.go`
- Clean up `DatabaseTableConfigSetter` interface if it references old type

Files:
- `internal/infrastructure/registry/database.go:82-256` (struct + defaults removed)
- `internal/composition/config/application.go` (**delete entire file**)
- `internal/composition/options/infrastructure/database.go:231-344` (rewrite or remove)
- `registry/registry.go` (clean up re-exports)

### Phase 7: Verification

- `go build ./...` — full package build
- `go vet ./...` — static analysis
- Verify no remaining `DatabaseTableConfig` struct references: `grep -r 'DatabaseTableConfig' packages/espyna-golang/`
- Verify `TableConfig.TableName()` used everywhere: `grep -r 'TableName(' packages/espyna-golang/`
- Build all apps: `cd apps/retail-admin && go build -tags "google_uuidv7,mock_auth,mock_storage,noop,postgresql,vanilla" ./...`

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang/registry/entityid/entityid.go` | Add domain slices + `All` | 1 |
| `packages/espyna-golang/internal/infrastructure/registry/database.go` | Add `TableConfig` type (P2), remove old struct (P6) | 2, 6 |
| `packages/espyna-golang/registry/registry.go` | Add new re-exports (P2), remove old (P6) | 2, 6 |
| `packages/espyna-golang/internal/composition/providers/domain/entity.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/product.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/subscription.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/common.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/event.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/workflow.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/revenue.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/inventory.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/expenditure.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/integration.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/ledger.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/treasury.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/domain/operation.go` | Signature + callsites | 3 |
| `packages/espyna-golang/internal/composition/providers/manager.go` | Type change + accessor | 4 |
| `packages/espyna-golang/internal/composition/core/container.go` | Type references | 4 |
| `packages/espyna-golang/internal/composition/core/usecases.go` | Pass new type to domain providers | 4 |
| `packages/espyna-golang/contrib/postgres/internal/adapter/adapter.go` | Rewrite `buildPgTableConfig()` | 5 |
| `packages/espyna-golang/internal/composition/config/application.go` | **Delete** | 6 |
| `packages/espyna-golang/internal/composition/options/infrastructure/database.go` | Rewrite table config functions | 6 |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** ~25 (already read during planning)
**Estimated files to modify:** 22
**Estimated context usage:** Medium (30-60 files)

**Sub-agent plan:**
- Phase 1 (entityid additions) — small, no agent needed
- Phase 2 (new type) — small, no agent needed
- Phase 3 (14 domain providers) — use sub-agent, repetitive mechanical changes
- Phase 4 (manager/container) — small, no agent needed
- Phase 5 (provider builders) — use sub-agent, can run in parallel with Phase 3
- Phase 6 (cleanup) — depends on Phases 3-5
- Phase 7 (verification) — depends on Phase 6

**Parallelism:** Phases 3 and 5 can run concurrently after Phase 2. Phase 4 can run after Phase 2 and before Phase 3 starts consuming the new type.

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| `config.DatabaseTableConfig` used in options path | Medium — could break option-based container init | Investigate `DatabaseTableConfigSetter` interface; update or remove |
| `entityid.All` slice missing entries | Low — env var override won't be scanned for that entity | Unit test that `len(All) == N` matches known count |
| Provider builders that don't iterate `All` | Low — those entities just use defaults (which is correct) | Only affects env var override scanning |
| Firestore/mock builders may have different patterns | Medium — need investigation | Read those files in Phase 5 before modifying |

**Dependencies:**
- Phase 1 must complete first (All slice needed by Phase 5)
- Phase 2 must complete before Phase 3-4 (new type must exist)
- Phase 3-5 can run in parallel
- Phase 6 depends on Phases 3-5
- Phase 7 depends on Phase 6

---

## Acceptance Criteria

- [ ] `entityid.All` slice exists with all 80+ entity constants
- [ ] `TableConfig` type exists with `TableName(entity string) string` method
- [ ] `NewTableConfig(prefix, overrides)` and `NewDefaultTableConfig()` constructors work
- [ ] All 14 domain providers use `tableConfig.TableName(entityid.X)` pattern
- [ ] PostgreSQL builder uses `entityid.All` loop instead of 100+ field assignments
- [ ] `DatabaseTableConfig` struct fully removed (zero references)
- [ ] `config/application.go` deleted
- [ ] `go build ./...` passes
- [ ] All 3 apps build successfully
- [ ] 8 previously-orphaned entities (JobActivity, ActivityLabor, etc.) automatically work as table names

---

## Design Decisions

**Why map-based instead of keeping the struct:**
The struct exists solely to hold string values that are (by default) identical to the entityid constant. A map eliminates O(N) maintenance cost per new entity. The only sacrifice is IDE autocompletion on field names, but the domain providers already use `entityid.X` constants which provide the same compile-time safety.

**Why `entityid.All` instead of reflection:**
Go reflection on constants is impossible (constants don't exist at runtime). We need an explicit slice to iterate. Domain-level sub-slices (`EntityEntities`, `ProductEntities`, etc.) keep it organized. The `All` slice is derived from sub-slices, so adding a new constant to a domain group automatically includes it in `All`.

**Why not code generation:**
The entityid package changes infrequently (new entities are added ~monthly). The maintenance cost of `go generate` tooling exceeds the cost of adding one constant + one slice entry. If entity count exceeds ~200, reconsider.

**Why delete `config/application.go` instead of unifying:**
It's a stale duplicate with fewer fields than `registry.DatabaseTableConfig`, and it defines a completely different Go type. The options system (`options/infrastructure/database.go`) that depends on it uses `LEAPFOR_DATABASE_*` env vars — a legacy prefix that should be migrated to the `POSTGRES_TABLE_*` convention used by the active code path. Unifying would add complexity for no benefit; deletion is cleaner.

**Why `TableName()` instead of `Get()`:**
`tableConfig.TableName(entityid.Client)` reads naturally in the domain context ("get the table name for the client entity"). `Get` is too generic and doesn't communicate intent.
