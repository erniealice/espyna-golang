# Entity ID Constants — Design Plan

**Date:** 2026-03-14
**Branch:** `dev/20260314-entity-id-constants`
**Status:** Draft
**Package:** espyna-golang-ryta

---

## Overview

Add a provider-agnostic `registry/entityid` package containing compile-time constants for all ~51 entity keys used in `RegisterRepositoryFactory` / `CreateRepository` calls. Then migrate all 3 database providers (postgresql, firestore, mock) and all consumer sites to use these constants instead of hardcoded strings.

---

## Motivation

- **Typo risk:** A misspelled string like `"clent"` is a silent runtime bug; `entityid.Client` is a compile error.
- **LLM maintainability:** A central catalog lets AI agents audit, generate, and validate entity registrations from one file.
- **Cross-provider consistency:** All 3 providers (postgresql, firestore, mock) register the same 51 entity keys — a shared constant set makes this contract explicit.
- **Refactoring safety:** Renaming an entity key becomes a single-constant change; the compiler flags every usage.

---

## Architecture

### New package

```
packages/espyna-golang-ryta/registry/entityid/
└── entityid.go          ← const-only, zero imports
```

**Module path:** `github.com/erniealice/espyna-golang/registry/entityid`

### Dependency direction

```
registry/entityid/  (leaf — no imports, only const declarations)
    ↑ imported by
contrib/postgres/internal/adapter/**    (register side)
contrib/google/internal/database/**     (register side)
internal/.../mock/**                    (register side)
internal/composition/providers/domain/* (consumer side)
```

**No circular dependency risk** — `registry/entityid` is a pure leaf package with zero imports. It sits alongside `registry/registry.go` in the public API surface but has no dependency on it.

### Constant naming convention

```go
package entityid

// Entity domain
const (
    Admin             = "admin"
    Client            = "client"
    ClientAttribute   = "client_attribute"
    ClientCategory    = "client_category"
    // ...
)
```

Group constants by domain with comments matching the existing `DatabaseTableConfig` groupings in `internal/infrastructure/registry/database.go:84-131`.

---

## Implementation Steps

### Phase 1: Create `registry/entityid` package

- Create `packages/espyna-golang-ryta/registry/entityid/entityid.go` with all 51 entity key constants
- Constants grouped by domain: Common, Entity, Event, Product, Subscription, Revenue, Expenditure, Inventory, Treasury, Ledger, Integration, Workflow
- Verify build: `cd packages/espyna-golang-ryta && go build ./registry/entityid/`

### Phase 2: Migrate PostgreSQL adapters (69 files)

Replace hardcoded strings in all `RegisterRepositoryFactory` calls:

```go
// Before
registry.RegisterRepositoryFactory("postgresql", "client", func(...) { ... })

// After
registry.RegisterRepositoryFactory("postgresql", entityid.Client, func(...) { ... })
```

Files organized by subdirectory:
- `contrib/postgres/internal/adapter/common/` — 1 file (category)
- `contrib/postgres/internal/adapter/attribute_value/` — 1 file
- `contrib/postgres/internal/adapter/entity/` — 17 files
- `contrib/postgres/internal/adapter/event/` — 2 files
- `contrib/postgres/internal/adapter/product/` — 11 files
- `contrib/postgres/internal/adapter/product_option/` — 1 file
- `contrib/postgres/internal/adapter/product_option_value/` — 1 file
- `contrib/postgres/internal/adapter/product_variant/` — 1 file
- `contrib/postgres/internal/adapter/product_variant_image/` — 1 file
- `contrib/postgres/internal/adapter/product_variant_option/` — 1 file
- `contrib/postgres/internal/adapter/revenue/` — 1 file
- `contrib/postgres/internal/adapter/revenue_attribute/` — 1 file
- `contrib/postgres/internal/adapter/revenue_category/` — 1 file
- `contrib/postgres/internal/adapter/revenue_line_item/` — 1 file
- `contrib/postgres/internal/adapter/expenditure/` — 4 files
- `contrib/postgres/internal/adapter/inventory_*/` — 6 files
- `contrib/postgres/internal/adapter/subscription/` — 12 files
- `contrib/postgres/internal/adapter/treasury/` — 2 files
- `contrib/postgres/internal/adapter/ledger/` — 1 file (document_template)
- `contrib/postgres/internal/adapter/document/` — 2 files (template, attachment)
- `contrib/postgres/internal/adapter/integrations/` — 1 file

**Each file change:** Add import `entityid "github.com/erniealice/espyna-golang/registry/entityid"`, replace string literal with constant.

### Phase 3: Migrate Firestore adapters (53 files)

Same pattern as Phase 2 but in `contrib/google/internal/database/firestore/`.

### Phase 4: Migrate Mock adapters (43 files)

Same pattern as Phase 2 but in `internal/infrastructure/adapters/secondary/database/mock/`.

### Phase 5: Migrate consumer sites (12 files)

Replace hardcoded entity key strings in `CreateRepository` / `GetRepositoryFactory` calls:

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

### Phase 6: Verification

- `go build ./...` — full package build
- `go vet ./...` — static analysis
- Verify no remaining hardcoded entity strings: `grep -r 'RegisterRepositoryFactory.*"postgresql".*"[a-z]' contrib/postgres/`
- Verify no remaining consumer strings: `grep -r 'CreateRepository.*"[a-z]' internal/composition/`
- Run existing tests if any

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang-ryta/registry/entityid/entityid.go` | **New file** — 51 entity key constants | 1 |
| `packages/espyna-golang-ryta/contrib/postgres/internal/adapter/**/*.go` | Replace string → constant (69 files) | 2 |
| `packages/espyna-golang-ryta/contrib/google/internal/database/firestore/**/*.go` | Replace string → constant (53 files) | 3 |
| `packages/espyna-golang-ryta/internal/infrastructure/adapters/secondary/database/mock/**/*.go` | Replace string → constant (43 files) | 4 |
| `packages/espyna-golang-ryta/internal/composition/providers/domain/*.go` | Replace string → constant (12 files) | 5 |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** ~30 (sample from each provider + consumers)
**Estimated files to modify:** 178 (1 new + 69 + 53 + 43 + 12)
**Estimated context usage:** High (60+ files)

**Sub-agent plan:**
- Phase 1 is small — single file creation, no agent needed
- Phase 2 (postgres, 69 files) — use sub-agent with `run_in_background: true`
- Phase 3 (firestore, 53 files) — use sub-agent with `run_in_background: true`, can run in parallel with Phase 2
- Phase 4 (mock, 43 files) — use sub-agent with `run_in_background: true`, can run in parallel with Phase 2-3
- Phase 5 (consumers, 12 files) — depends on Phase 1 (constants must exist), but NOT on Phase 2-4
- Phase 6 (verification) — depends on all previous phases

**Parallelism:** Phases 2, 3, 4, and 5 can all run concurrently after Phase 1 completes.

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing entity key in constants | Build failure in adapters that reference it | Derive full list from grep of all 3 providers |
| Entity key mismatch between constant value and existing string | Silent runtime failure | Verify each `const X = "x"` matches the exact string currently used |
| New adapters added during implementation | New files still use raw strings | Update contributor docs to reference entityid package |

**Dependencies:**
- Phase 1 must complete before Phases 2-5 (constants must exist to import)
- Phases 2, 3, 4, 5 are independent of each other and can run in parallel
- Phase 6 depends on all phases

---

## Acceptance Criteria

- [ ] `registry/entityid/entityid.go` exists with all 51 entity constants
- [ ] Zero hardcoded entity key strings remain in `RegisterRepositoryFactory` calls (all 3 providers)
- [ ] Zero hardcoded entity key strings remain in `CreateRepository` calls (all consumer files)
- [ ] `go build ./...` passes with no errors
- [ ] `go vet ./...` passes with no warnings
- [ ] No circular dependency introduced (verified by successful build)
- [ ] Constants are grouped by domain with comments matching `DatabaseTableConfig` structure

---

## Design Decisions

**Why `registry/entityid/` and not `contrib/postgres/entities.go`:**
Codex CLI correctly identified that entity keys are provider-agnostic — the same `"client"` key is used across postgresql, firestore, and mock. Placing constants under a single provider would create wrong dependency direction. `registry/entityid/` is neutral, sits alongside the public registry API, and can be imported by any provider.

**Why not a typed enum:**
Go constants are simpler, require zero imports in the declaring package, and the registry API already accepts `string`. A custom type would require changes to `RegisterRepositoryFactory` and `CreateRepository` signatures — scope creep for no benefit.

**Why not generate from `DatabaseTableConfig`:**
The entity keys and table config field names are related but not identical (e.g., `IntegrationPayment` field vs `"integration_payment"` key). Code generation would add build complexity. Manual constants with a verification grep in Phase 6 is simpler and sufficient.
