# Espyna Sub-Module Split — Design Plan

**Date:** 2026-03-06
**Branch:** `dev/20260306-contrib-split`
**Status:** In Progress (Phases 0-5 complete, Phase 6 in progress)
**Package:** espyna-golang-ryta

---

## Overview

Split heavy SDK adapters out of the core espyna-golang module into `contrib/` sub-modules, each with its own `go.mod`. This eliminates ~40 transitive cloud SDK dependencies (Azure, AWS, Google Cloud) from consumer apps that don't use them.

---

## Motivation

`go mod tidy` is build-tag-agnostic by design. Even though espyna uses `//go:build` tags to conditionally compile adapters, ALL cloud SDK dependencies flow into every consumer app's `go.mod` as transitive indirect deps. A service-admin app using only vanilla HTTP + PostgreSQL shouldn't need Azure, AWS, or Google Cloud SDKs in its dependency tree.

**Before:** `service-admin → espyna-golang` (one module, ALL SDKs — ~40 indirect cloud deps)
**After:** `service-admin → espyna-golang` (core, no cloud SDKs) + `→ contrib/postgres` (lib/pq only)

---

## Architecture

### What stays in core espyna-golang

| Category | Rationale |
|----------|-----------|
| Composition (container, contracts, routing, providers, options) | Zero external SDK deps |
| Application (usecases, ports, shared context) | Zero external SDK deps |
| Registry (factory registration framework) | Zero external SDK deps |
| Mock/noop/local adapters | Zero external deps |
| Vanilla HTTP adapter | stdlib only |
| Payment adapters (paypal, asiapay, maya) | Custom HTTP clients, no SDK |
| Scheduler adapters (calendly) | Custom HTTP client, no SDK |
| Microsoft email adapter | Custom HTTP client, no SDK |
| JWT auth adapter | stdlib only |
| Translation adapters (file, lyngua, noop, mock) | Lightweight deps |
| ID adapters (noop, uuidv7) | google/uuid is lightweight |
| ~~Gin/Fiber HTTP adapters~~ | ~~Deferred~~ — Moved to Phase 6 (in progress) |

### What moves to contrib sub-modules

| Module | Contents | Files | SDK Deps Isolated |
|--------|----------|-------|-------------------|
| `contrib/postgres` | PostgreSQL adapter (68 files), ledger, reference/checker | ~72 | lib/pq, golang-migrate |
| `contrib/google` | Firebase auth, Firestore DB, GCS storage, Gmail, Sheets, GCP/Firebase common | ~75 | cloud.google.com/go/*, firebase.google.com/* |
| `contrib/azure` | Azure Blob Storage adapter | ~2 | github.com/Azure/azure-sdk-for-go/* |
| `contrib/aws` | AWS S3 Storage adapter | ~2 | github.com/aws/aws-sdk-go-v2/* |

### Key design properties enabling the split

1. **Self-registration pattern**: Adapters use `init()` + generic `FactoryRegistry` — no hardcoded adapter imports in core
2. **Clean ports**: `internal/application/ports/` has zero external SDK deps (only esqyma protobuf)
3. **Clean registry**: `internal/infrastructure/registry/` is build-tag-free, uses Go generics
4. **Domain packages unaffected**: centymo, entydad, fycha don't import espyna at all
5. **Apps use only consumer package**: `consumer.NewContainerFromEnv()` is the sole entry point

---

## Implementation Steps

### Phase 0: Export Core Packages (prerequisite for all contrib modules)

Contrib modules can't import `internal/` packages across module boundaries. Create **type alias re-export packages** that expose internal interfaces publicly without breaking existing internal consumers.

**0.1 Create `ports/` re-export package:**
- New file: `packages/espyna-golang-ryta/ports/ports.go`
- Pattern: `type DatabaseProvider = ports.DatabaseProvider` for all port interfaces
- Source: `internal/application/ports/exports.go` (362 lines of re-exports)

**0.2 Create `registry/` re-export package:**
- New file: `packages/espyna-golang-ryta/registry/registry.go`
- Re-export: `FactoryRegistry`, all `Register*`/`Get*`/`Build*` functions from `internal/infrastructure/registry/`
- Re-export: `DatabaseTableConfig`, `RepositoryFactory`, `DatabaseOperationsFactory`, all related functions

**0.3 Create `database/interfaces/` re-export package:**
- New file: `packages/espyna-golang-ryta/database/interfaces/interfaces.go`
- Source: `internal/infrastructure/adapters/secondary/database/common/interface/` (operations.go, query.go, transactions.go)
- Re-export: `DatabaseOperation`, `TransactionAware`, `QueryBuilder`, `QueryFilter`, `Transaction`, `TransactionManager`, etc.

**0.4 Create `database/model/` re-export package:**
- New file: `packages/espyna-golang-ryta/database/model/model.go`
- Source: `internal/infrastructure/adapters/secondary/database/common/model/`
- Re-export: base models, errors, validation types

**0.5 Create `database/operations/` re-export package:**
- New file: `packages/espyna-golang-ryta/database/operations/operations.go`
- Source: `internal/infrastructure/adapters/secondary/database/common/operations/` (context.go, helpers.go, query_builder.go)
- Re-export: `QueryBuilder` implementation, helpers

**0.6 Create `database/transactions/` re-export package:**
- New file: `packages/espyna-golang-ryta/database/transactions/transactions.go`
- Source: `internal/infrastructure/adapters/secondary/database/common/transactions/service_adapter.go`

**0.7 Create `storage/helpers/` re-export package:**
- New file: `packages/espyna-golang-ryta/storage/helpers/helpers.go`
- Source: `internal/infrastructure/adapters/secondary/storage/common/helpers.go`
- Re-export: `GenerateObjectID`, `DetectContentType`

**0.8 Add ledger registry support:**
- Add `RegisterLedgerReportingFactory` / `GetLedgerReportingFactory` to `internal/infrastructure/registry/ledger.go`
- Replace build-tagged `consumer/adapter_ledger_postgres.go` + `adapter_ledger_noop.go` with unified `consumer/adapter_ledger_registry.go` that uses registry discovery (import = opt-in, no build tags needed)

**0.9 Verify:** `go build ./ports/... ./registry/... ./database/... ./storage/...`

---

### Phase 1: contrib/postgres

**1.1 Create sub-module structure:**
- New: `packages/espyna-golang-ryta/contrib/postgres/go.mod`
  ```
  module github.com/erniealice/espyna-golang/contrib/postgres
  require (
      github.com/erniealice/espyna-golang v0.0.0
      github.com/erniealice/esqyma v0.0.0
      github.com/lib/pq v1.10.9
      github.com/golang-migrate/migrate/v4 v4.19.0
  )
  replace github.com/erniealice/espyna-golang => ../..
  replace github.com/erniealice/esqyma => ../../../esqyma-ryta
  ```

**1.2 Move adapter code (68 files):**
- `internal/infrastructure/adapters/secondary/database/postgres/` → `contrib/postgres/internal/adapter/`
- Preserve entire directory structure under `internal/adapter/` (entity/, product/, revenue/, ledger/, etc.)

**1.3 Move consumer registration & ledger:**
- `consumer/register_database_postgres.go` logic → `contrib/postgres/register.go` (init() function that calls registry registration)
- `consumer/adapter_ledger_postgres.go` → `contrib/postgres/ledger.go` (exported `NewLedgerReportingService` factory)

**1.4 Move reference checker:**
- `reference/checker.go` → `contrib/postgres/reference/checker.go`

**1.5 Update imports in all moved files:**
- `internal/application/ports` → `github.com/erniealice/espyna-golang/ports`
- `internal/infrastructure/registry` → `github.com/erniealice/espyna-golang/registry`
- `internal/.../database/common/interface` → `github.com/erniealice/espyna-golang/database/interfaces`
- `internal/.../database/common/model` → `github.com/erniealice/espyna-golang/database/model`
- `internal/.../database/common/operations` → `github.com/erniealice/espyna-golang/database/operations`
- `internal/.../database/common/transactions` → `github.com/erniealice/espyna-golang/database/transactions`
- Intra-postgres: adjust to `contrib/postgres/internal/adapter/...`

**1.6 Remove build tags from moved files** (import = opt-in, no `//go:build postgresql` needed)

**1.7 Remove from consumer:**
- Delete `consumer/register_database_postgres.go`
- Delete `consumer/adapter_ledger_postgres.go`
- Delete `consumer/adapter_ledger_noop.go`

**1.8 Update go.work:**
- Add `./packages/espyna-golang-ryta/contrib/postgres`

**1.9 Update consumer apps** (all 4: retail-admin, retail-client, service-admin, service-client):
- Add: `import _ "github.com/erniealice/espyna-golang/contrib/postgres"` in container.go
- service-admin: Change `"github.com/erniealice/espyna-golang/reference"` → `"github.com/erniealice/espyna-golang/contrib/postgres/reference"` in views.go

**1.10 Update app go.mod files:**
- Add `require github.com/erniealice/espyna-golang/contrib/postgres v0.0.0`
- Add `replace github.com/erniealice/espyna-golang/contrib/postgres => ../../packages/espyna-golang-ryta/contrib/postgres`

**1.11 Update run.ps1 scripts:**
- Remove `postgresql` from tag generation (line ~36 in both scripts)
- Postgres is now opt-in via import, not build tag

**1.12 Verify:**
- `go build ./...` in core espyna
- `go build ./...` in contrib/postgres
- `go build -tags "google_uuidv7,mock_auth,mock_storage,noop,vanilla,lyngua" ./...` in service-admin
- E2E tests pass

---

### Phase 2: contrib/google

**2.1 Create sub-module:**
- New: `packages/espyna-golang-ryta/contrib/google/go.mod` with Google Cloud + Firebase SDK deps

**2.2 Move packages (75 files):**
- `secondary/common/gcp/` → `contrib/google/internal/common/gcp/`
- `secondary/common/firebase/` → `contrib/google/internal/common/firebase/`
- `secondary/common/google/` → `contrib/google/internal/common/google/`
- `secondary/auth/firebase/` → `contrib/google/internal/auth/firebase/`
- `secondary/database/firestore/` (61 files) → `contrib/google/internal/database/firestore/`
- `secondary/storage/gcs/` → `contrib/google/internal/storage/gcs/`
- `secondary/email/gmail/` → `contrib/google/internal/email/gmail/`
- `secondary/tabular/googlesheets/` → `contrib/google/internal/tabular/googlesheets/`

**2.3 Create `contrib/google/register.go`** with init() importing all Google adapters

**2.4 Remove from consumer:**
- `register_auth_firebase.go`
- `register_database_firestore.go`
- `register_storage_gcs.go`
- `register_email_gmail.go`
- `register_tabular_googlesheets.go`

**2.5 Update imports in all moved files** (same pattern as Phase 1)

**2.6 Remove build tags from moved files**

**2.7 Update go.work:** Add `./packages/espyna-golang-ryta/contrib/google`

**2.8 Verify:** `go build ./...` in core espyna + contrib/google

---

### Phase 3: contrib/azure

**3.1 Create sub-module:**
- New: `packages/espyna-golang-ryta/contrib/azure/go.mod` with Azure SDK deps

**3.2 Move:** `secondary/storage/azure/adapter.go` → `contrib/azure/internal/adapter/adapter.go`

**3.3 Create `contrib/azure/register.go`** with init()

**3.4 Remove:** `consumer/register_storage_azure.go`

**3.5 Update imports, remove build tags**

**3.6 Update go.work:** Add `./packages/espyna-golang-ryta/contrib/azure`

---

### Phase 4: contrib/aws

**4.1 Create sub-module:**
- New: `packages/espyna-golang-ryta/contrib/aws/go.mod` with AWS SDK deps

**4.2 Move:** `secondary/storage/s3/adapter.go` → `contrib/aws/internal/adapter/adapter.go`

**4.3 Create `contrib/aws/register.go`** with init()

**4.4 Remove:** `consumer/register_storage_s3.go`

**4.5 Update imports, remove build tags**

**4.6 Update go.work:** Add `./packages/espyna-golang-ryta/contrib/aws`

---

### Phase 5: Clean Up Core go.mod

**5.1 Remove cloud SDK deps from `packages/espyna-golang-ryta/go.mod`:**
- All `cloud.google.com/go/*` (15+ packages)
- All `firebase.google.com/*`
- All `google.golang.org/api/*`
- All `github.com/Azure/*` (4 packages)
- All `github.com/aws/*` (14 packages)
- `github.com/golang-migrate/migrate/v4`
- `github.com/lib/pq`

**5.2 Run `go mod tidy`** on core espyna, each contrib module, and each app

**5.3 Verify:** service-admin's `go.mod` should have ~60% fewer indirect deps

---

### Phase 6: contrib/gin, contrib/fiber

HTTP framework adapters that pull Gin (~6 indirect deps) and Fiber (~6 indirect deps including fasthttp, quic-go) into every app's go.mod even when the app uses `vanilla` HTTP.

**Challenge:** These adapters import composition internals (`contracts`, `routing`, `core`) which are NOT yet in re-export packages. Phase 6 requires creating additional re-export packages for composition types.

#### Phase 6.0: Export Composition Packages (prerequisite)

Create re-export packages for composition internals needed by HTTP adapters:

**6.0.1 Create `composition/contracts/` re-export package:**
- Source: `internal/composition/contracts/` (HTTPAdapter, RouteRegistrar, Middleware, etc.)
- These are the core interfaces that HTTP adapters implement

**6.0.2 Create `composition/routing/` re-export package:**
- Source: `internal/composition/routing/` (Router, RouteConfig, route builder functions)
- Used by adapters to register routes

**6.0.3 Create `composition/core/` re-export package (if needed):**
- Source: `internal/composition/core/` (Container, Provider types)
- May not be needed if adapters only depend on contracts/routing

**6.0.4 Verify:** `go build ./composition/...` passes

#### Phase 6a: contrib/gin

**6a.1 Create sub-module:**
- New: `packages/espyna-golang-ryta/contrib/gin/go.mod`
  ```
  module github.com/erniealice/espyna-golang/contrib/gin
  require (
      github.com/erniealice/espyna-golang v0.0.0
      github.com/gin-gonic/gin v1.11.0
      github.com/gin-contrib/cors v1.7.6
      github.com/gin-contrib/gzip v1.2.3
  )
  replace github.com/erniealice/espyna-golang => ../..
  ```

**6a.2 Identify and move Gin adapter files:**
- `consumer/register_server_gin.go` → `contrib/gin/register.go`
- `consumer/server_gin.go` → `contrib/gin/internal/adapter/server.go` (if exists)
- Any files with `//go:build gin` in `internal/composition/` or `internal/infrastructure/adapters/`

**6a.3 Update imports to use composition re-export packages**

**6a.4 Remove `//go:build gin` tags from moved files**

**6a.5 Update go.work:** Add `./packages/espyna-golang-ryta/contrib/gin`

**6a.6 Remove from consumer:** `register_server_gin.go`

**6a.7 Verify:** `go build ./...` in contrib/gin + all apps still build

#### Phase 6b: contrib/fiber

**6b.1 Create sub-module:**
- New: `packages/espyna-golang-ryta/contrib/fiber/go.mod`
  ```
  module github.com/erniealice/espyna-golang/contrib/fiber
  require (
      github.com/erniealice/espyna-golang v0.0.0
      github.com/gofiber/fiber/v2 v2.52.9
      github.com/gofiber/fiber/v3 v3.0.0-rc.2
  )
  replace github.com/erniealice/espyna-golang => ../..
  ```

**6b.2 Identify and move Fiber adapter files:**
- `consumer/register_server_fiber.go` → `contrib/fiber/register.go`
- `consumer/register_server_fiberv3.go` → `contrib/fiber/register_v3.go`
- Any files with `//go:build fiber` or `//go:build fiberv3` in composition/infrastructure

**6b.3 Update imports, remove build tags**

**6b.4 Update go.work:** Add `./packages/espyna-golang-ryta/contrib/fiber`

**6b.5 Remove from consumer:** `register_server_fiber.go`, `register_server_fiberv3.go`

**6b.6 Verify:** `go build ./...` in contrib/fiber + all apps still build

#### Phase 6c: Clean Up Core go.mod

**6c.1 Remove HTTP framework deps from core go.mod:**
- `github.com/gin-gonic/gin` + `github.com/gin-contrib/*` (3 packages)
- `github.com/gofiber/fiber/v2` + `github.com/gofiber/fiber/v3` + `github.com/gofiber/*` (4 packages)
- Transitive deps: `valyala/fasthttp`, `quic-go/*`, `bytedance/sonic`, etc.

**6c.2 Run `go mod tidy`** on core + contrib/gin + contrib/fiber + all apps

**6c.3 Verify:** service-admin `go.mod` should now only have:
- Domain packages (centymo, entydad, fycha, esqyma, lyngua, pyeza)
- contrib/postgres
- gRPC/protobuf (needed by esqyma)
- CEL (needed by RBAC)
- stdlib extensions (golang.org/x/*)
- Truly minimal set — zero framework AND zero cloud SDK deps

#### Expected outcome

| Metric | Before Phase 6 | After Phase 6 |
|--------|----------------|---------------|
| Core go.mod indirect deps | ~72 | ~40-50 (estimated) |
| Gin deps in service-admin | 6 packages | 0 |
| Fiber deps in service-admin | 6 packages | 0 |
| Core direct deps | 14 | 8-10 (gin, fiber, their contrib removed) |

#### Risk: Composition coupling depth

The critical question is how deeply Gin/Fiber adapters couple to composition internals. If they only use `contracts.HTTPAdapter` and `routing.Router`, it's straightforward. If they reach into `core.Container` internals, it may require exporting too many types. The `http-framework-builder` agent is currently assessing this.

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang-ryta/ports/ports.go` | **New** — re-export `internal/application/ports` | 0 |
| `packages/espyna-golang-ryta/registry/registry.go` | **New** — re-export `internal/infrastructure/registry` | 0 |
| `packages/espyna-golang-ryta/database/interfaces/interfaces.go` | **New** — re-export database interfaces | 0 |
| `packages/espyna-golang-ryta/database/model/model.go` | **New** — re-export database models | 0 |
| `packages/espyna-golang-ryta/database/operations/operations.go` | **New** — re-export query builder | 0 |
| `packages/espyna-golang-ryta/database/transactions/transactions.go` | **New** — re-export transaction types | 0 |
| `packages/espyna-golang-ryta/storage/helpers/helpers.go` | **New** — re-export storage helpers | 0 |
| `packages/espyna-golang-ryta/internal/infrastructure/registry/ledger.go` | **New** — ledger registry functions | 0 |
| `packages/espyna-golang-ryta/consumer/adapter_ledger.go` | Rewrite — use registry discovery instead of build tags | 0 |
| `packages/espyna-golang-ryta/consumer/adapter_ledger_postgres.go` | **Delete** — moves to contrib/postgres | 1 |
| `packages/espyna-golang-ryta/consumer/adapter_ledger_noop.go` | **Delete** — replaced by registry-based adapter | 0 |
| `packages/espyna-golang-ryta/contrib/postgres/go.mod` | **New** — sub-module definition | 1 |
| `packages/espyna-golang-ryta/contrib/postgres/register.go` | **New** — init() registration | 1 |
| `packages/espyna-golang-ryta/contrib/postgres/ledger.go` | **New** — exported ledger factory | 1 |
| `packages/espyna-golang-ryta/contrib/postgres/reference/checker.go` | **Move** from `reference/checker.go` | 1 |
| `packages/espyna-golang-ryta/contrib/postgres/internal/adapter/` | **Move** — 68 files from postgres adapter dir | 1 |
| `packages/espyna-golang-ryta/consumer/register_database_postgres.go` | **Delete** | 1 |
| `packages/espyna-golang-ryta/reference/checker.go` | **Delete** (moved) | 1 |
| `apps/retail-admin/internal/composition/container.go` | Add contrib/postgres import | 1 |
| `apps/retail-client/internal/composition/container.go` | Add contrib/postgres import | 1 |
| `apps/service-admin/internal/composition/container.go` | Add contrib/postgres import | 1 |
| `apps/service-admin/internal/composition/views.go` | Update reference import path | 1 |
| `apps/service-client/internal/composition/container.go` | Add contrib/postgres import | 1 |
| `apps/retail-admin/go.mod` | Add contrib/postgres require+replace | 1 |
| `apps/retail-client/go.mod` | Add contrib/postgres require+replace | 1 |
| `apps/service-admin/go.mod` | Add contrib/postgres require+replace | 1 |
| `apps/service-client/go.mod` | Add contrib/postgres require+replace | 1 |
| `apps/service-admin/scripts/run.ps1` | Remove postgresql from tag generation | 1 |
| `apps/retail-admin/scripts/run.ps1` | Remove postgresql from tag generation | 1 |
| `go.work` | Add contrib/postgres, contrib/google, contrib/azure, contrib/aws | 1-4 |
| `packages/espyna-golang-ryta/contrib/google/go.mod` | **New** — sub-module definition | 2 |
| `packages/espyna-golang-ryta/contrib/google/register.go` | **New** — init() registration | 2 |
| `packages/espyna-golang-ryta/contrib/google/internal/` | **Move** — ~75 files from Google adapters | 2 |
| `packages/espyna-golang-ryta/consumer/register_auth_firebase.go` | **Delete** | 2 |
| `packages/espyna-golang-ryta/consumer/register_database_firestore.go` | **Delete** | 2 |
| `packages/espyna-golang-ryta/consumer/register_storage_gcs.go` | **Delete** | 2 |
| `packages/espyna-golang-ryta/consumer/register_email_gmail.go` | **Delete** | 2 |
| `packages/espyna-golang-ryta/consumer/register_tabular_googlesheets.go` | **Delete** | 2 |
| `packages/espyna-golang-ryta/contrib/azure/go.mod` | **New** — sub-module definition | 3 |
| `packages/espyna-golang-ryta/contrib/azure/register.go` | **New** — init() registration | 3 |
| `packages/espyna-golang-ryta/contrib/azure/internal/adapter/adapter.go` | **Move** from azure storage adapter | 3 |
| `packages/espyna-golang-ryta/consumer/register_storage_azure.go` | **Delete** | 3 |
| `packages/espyna-golang-ryta/contrib/aws/go.mod` | **New** — sub-module definition | 4 |
| `packages/espyna-golang-ryta/contrib/aws/register.go` | **New** — init() registration | 4 |
| `packages/espyna-golang-ryta/contrib/aws/internal/adapter/adapter.go` | **Move** from s3 storage adapter | 4 |
| `packages/espyna-golang-ryta/consumer/register_storage_s3.go` | **Delete** | 4 |
| `packages/espyna-golang-ryta/go.mod` | Remove all cloud SDK deps | 5 |
| `packages/espyna-golang-ryta/composition/contracts/contracts.go` | **New** — re-export composition contracts | 6.0 |
| `packages/espyna-golang-ryta/composition/routing/routing.go` | **New** — re-export routing types | 6.0 |
| `packages/espyna-golang-ryta/contrib/gin/go.mod` | **New** — sub-module definition | 6a |
| `packages/espyna-golang-ryta/contrib/gin/register.go` | **New** — init() registration | 6a |
| `packages/espyna-golang-ryta/contrib/gin/internal/adapter/` | **Move** — Gin adapter files | 6a |
| `packages/espyna-golang-ryta/consumer/register_server_gin.go` | **Delete** | 6a |
| `packages/espyna-golang-ryta/contrib/fiber/go.mod` | **New** — sub-module definition | 6b |
| `packages/espyna-golang-ryta/contrib/fiber/register.go` | **New** — init() registration | 6b |
| `packages/espyna-golang-ryta/contrib/fiber/internal/adapter/` | **Move** — Fiber adapter files | 6b |
| `packages/espyna-golang-ryta/consumer/register_server_fiber.go` | **Delete** | 6b |
| `packages/espyna-golang-ryta/consumer/register_server_fiberv3.go` | **Delete** | 6b |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** ~100 (registry, ports, consumer, adapter entry points, app containers, go.mod files)
**Estimated files to modify/create/move:** ~160+ (68 postgres move, 75 google move, ~8 new re-export pkgs, ~15 app-level changes)
**Estimated context usage:** High (160+ files)

**Sub-agent plan:**
- Phase 0 (re-export packages) can run in a single session — mostly boilerplate type aliases
- Phase 1 (postgres) is the largest and most critical — use background agents to:
  - Agent A: Move 68 postgres files + update imports
  - Agent B: Update 4 app container.go + go.mod files
- Phase 2 (google) — independent, can run in parallel with Phase 1 verification
- Phase 3+4 (azure, aws) — small, sequential, ~5 min each
- Phase 5 (cleanup) — depends on all prior phases

**Recommended: team of 3 agents** for Phases 1-2 (after Phase 0 completes):
1. **postgres-builder**: Phase 1 file moves + import updates
2. **google-builder**: Phase 2 file moves + import updates
3. **app-integrator**: App-level changes (container.go, go.mod, run.ps1) after builders finish

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| Module cycle (core ↔ contrib) | Build breaks | Strict rule: core NEVER imports contrib. Verify with `go mod graph` |
| 68 postgres files = large move | Git history noise | Move directory wholesale, then batch-update imports |
| Broken internal imports after move | Build breaks | Type alias re-export pattern preserves backward compat for core internal users |
| Build tag confusion | Runtime errors | Clear rule: contrib modules have NO build tags (import = opt-in). Core adapters keep existing tags |
| go.work sync | Build breaks | Add all contrib modules to go.work before running any `go mod tidy` |
| Ledger adapter build-tag removal | Runtime nil panic | Replace with registry-based discovery in Phase 0 before moving postgres |

**Dependencies:**
- Phase 0 must complete before Phases 1-4 (contrib modules need re-export packages)
- Phases 1-4 are independent of each other (can run in parallel)
- Phase 5 depends on all of Phases 1-4
- Phase 6 is deferred (no dependency)

---

## Acceptance Criteria

- [ ] `go build ./...` passes in core espyna module (no cloud SDK imports)
- [ ] `go build ./...` passes in each contrib module (postgres, google, azure, aws)
- [ ] `go build -tags "google_uuidv7,mock_auth,mock_storage,noop,vanilla,lyngua" ./...` passes in service-admin
- [ ] `go build -tags "google_uuidv7,mock_auth,mock_storage,noop,postgresql,vanilla" ./...` passes in retail-admin (backward compat during transition)
- [ ] `go mod tidy` on service-admin no longer pulls Azure/AWS/Google Cloud indirect deps
- [ ] service-admin `go.mod` has ~60% fewer indirect deps
- [ ] Existing E2E tests pass (77+ retail-admin, 61 service-admin)
- [ ] No module cycles: `go mod graph` shows no core → contrib edges
- [ ] `reference.Checker` works correctly from `contrib/postgres/reference` import path

---

## Design Decisions

### Why type alias re-exports instead of moving packages out of internal?

Moving `internal/application/ports` to `ports/` would break every internal import. Type aliases (`type X = internal.X`) let both paths coexist: internal code keeps using `internal/application/ports`, while contrib modules use the public `ports/` package. Both resolve to the same types at compile time.

### Why not just remove build tags without sub-modules?

Build tags serve a different purpose (conditional compilation for the same binary). The problem is module-level dependency resolution — `go mod tidy` sees all `.go` files regardless of tags. Sub-modules are the only way to truly isolate SDK dependencies at the module graph level.

### Why keep Gin/Fiber in core (Phase 6 deferred)?

These HTTP adapters import `internal/composition/contracts`, `internal/composition/routing`, and `internal/composition/core` — deeply coupled composition internals that would need their own re-export packages. The benefit is lower (web framework deps are smaller than cloud SDKs), and the current build-tag approach works well enough for framework selection.

### Why registry-based ledger instead of build tags?

The ledger adapter is a consumer-level concern (not an internal adapter) that bridges postgres-specific code. With contrib/postgres as a separate module, the build-tag approach breaks (the tag no longer controls compilation). Registry-based discovery (`RegisterLedgerReportingFactory` in contrib/postgres's `init()`, fallback to nil in core) is cleaner and consistent with how all other adapters work.
