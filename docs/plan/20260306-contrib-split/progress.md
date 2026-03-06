# Espyna Sub-Module Split — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-03-06
**Branch:** `development`
**Last Updated:** 2026-03-06 11:35

---

## Phase 0: Export Core Packages — COMPLETE

- [x] Create `ports/ports.go` — re-export all port interfaces (60+ types, 40+ entity constants, action constants, error constructors)
- [x] Create `registry/registry.go` — re-export FactoryRegistry generic type, all database/storage/auth/email/tabular registry functions
- [x] Create `database/interfaces/interfaces.go` — re-export DatabaseOperation, TransactionAware, QueryBuilder, Transaction, TransactionManager
- [x] Create `database/model/model.go` — re-export BaseModel, TransactionError, DatabaseError, ValidationError
- [x] Create `database/operations/operations.go` — re-export ProtobufTimestamp/Mapper, ConvertMapToProtobuf, ConvertSliceToProtobuf (generic function wrappers)
- [x] Create `database/transactions/transactions.go` — re-export TransactionServiceAdapter
- [x] Create `storage/helpers/helpers.go` — re-export GenerateObjectID, DetectContentType
- [x] Create `internal/infrastructure/registry/ledger.go` — new ledger reporting factory registry (sync.RWMutex)
- [x] Rewrite `consumer/adapter_ledger.go` — registry discovery replaces build-tagged postgres/noop variants
- [x] Delete `consumer/adapter_ledger_noop.go` and `consumer/adapter_ledger_postgres.go`
- [x] Verify: `go build ./ports/... ./registry/... ./database/... ./storage/...` — PASS

### Key Decisions
- Generic functions (`ConvertMapToProtobuf`, `ConvertSliceToProtobuf`, `NewFactoryRegistry`) wrapped as function declarations, not var aliases (Go limitation)
- Ledger registry uses `func(db any, config any) any` signature to avoid import cycles

---

## Phase 1: contrib/postgres — COMPLETE

- [x] Create `contrib/postgres/go.mod` — deps: lib/pq, golang-migrate, espyna core, esqyma
- [x] Move 68 postgres adapter files to `contrib/postgres/internal/adapter/`
- [x] Move migration service to `contrib/postgres/internal/migration/` (was missed initially — pulled golang-migrate into core)
- [x] Create `contrib/postgres/register.go` — blank import triggers init() registration
- [x] Create `contrib/postgres/ledger.go` — factory init() with reflect-based config extraction
- [x] Move `reference/checker.go` → `contrib/postgres/reference/checker.go`
- [x] Update all imports: ports, registry, database/interfaces, database/model, database/operations, database/transactions
- [x] Remove `//go:build postgresql` tags from all 68+ moved files
- [x] Add `use ./packages/espyna-golang-ryta/contrib/postgres` to `go.work`
- [x] Update 4 app container.go files — add `import _ "github.com/erniealice/espyna-golang/contrib/postgres"`
- [x] Update 4 app go.mod files — add require + replace for contrib/postgres
- [x] Verify: `go build ./...` in contrib/postgres — PASS

### Files Moved
- `internal/infrastructure/adapters/secondary/database/postgres/` → `contrib/postgres/internal/adapter/` (68 files)
- `internal/infrastructure/adapters/secondary/migration/postgres_migration_service.go` → `contrib/postgres/internal/migration/`
- `consumer/register_database_postgres.go` → logic in `contrib/postgres/register.go`
- `consumer/adapter_ledger_postgres.go` → `contrib/postgres/ledger.go`
- `reference/checker.go` (postgresql-tagged) → `contrib/postgres/reference/checker.go`

---

## Phase 2: contrib/google — COMPLETE

- [x] Create `contrib/google/go.mod` — deps: cloud.google.com/go/firestore, secretmanager, storage, firebase, oauth2, google.golang.org/api
- [x] Move ~75 files: GCP common, Firebase common, Google common, Firebase auth, Firestore DB, GCS storage, Gmail, Google Sheets
- [x] Create `contrib/google/register.go` — init() imports for all Google adapter packages
- [x] Fix replace directives: `leapfor.xyz/copya` needed 5 `../` levels (not 4) due to extra depth
- [x] Fix import alias issues: sed mangled grpc `codes`/`status` imports in firestore/adapter.go
- [x] Remove build tags from all moved files
- [x] Add `use ./packages/espyna-golang-ryta/contrib/google` to `go.work`
- [x] Verify: `go build ./...` in contrib/google — PASS (all 17 packages)

### Files Moved
- `secondary/common/gcp/` → `contrib/google/internal/common/gcp/`
- `secondary/common/firebase/` → `contrib/google/internal/common/firebase/`
- `secondary/common/google/` → `contrib/google/internal/common/google/`
- `secondary/auth/firebase/` → `contrib/google/internal/auth/firebase/`
- `secondary/database/firestore/` → `contrib/google/internal/database/firestore/` (14 sub-packages)
- `secondary/storage/gcs/` → `contrib/google/internal/storage/gcs/`
- `secondary/email/gmail/` → `contrib/google/internal/email/gmail/`
- `secondary/tabular/googlesheets/` → `contrib/google/internal/tabular/googlesheets/`
- Consumer files removed: `register_auth_firebase.go`, `register_database_firestore.go`, `register_storage_gcs.go`, `register_email_gmail.go`, `register_tabular_googlesheets.go`

### Gotcha
- `cmd/seeder/workflow_templates_http.go` directly imported `cloud.google.com/go/firestore` (bypassing adapters) — DELETED. The adapter-based `workflow_templates.go` remains.

---

## Phase 3: contrib/azure — COMPLETE

- [x] Create `contrib/azure/go.mod` with Azure SDK deps
- [x] Move azure storage adapter to `contrib/azure/internal/adapter/`
- [x] Create `contrib/azure/register.go` with init()
- [x] Fixed 9 Azure SDK compilation bugs hidden behind build tags (API changes in azblob v1.6.2)
- [x] Add `use ./packages/espyna-golang-ryta/contrib/azure` to `go.work`
- [x] Verify: `go build ./...` in contrib/azure — PASS

### Azure SDK Bugs Found & Fixed
1. `map[string]*string` metadata → conversion helpers added
2. `GetProperties` returns value types (not pointers)
3. `PublicAccessTypeNone` removed from Azure SDK
4. `PublicAccess` renamed to `BlobPublicAccess`
5-9. Various pointer/value type mismatches

---

## Phase 4: contrib/aws — COMPLETE

- [x] Create `contrib/aws/go.mod` with AWS SDK deps
- [x] Move s3 storage adapter to `contrib/aws/internal/adapter/`
- [x] Create `contrib/aws/register.go` with init()
- [x] Add `use ./packages/espyna-golang-ryta/contrib/aws` to `go.work`
- [x] Verify: `go build ./...` in contrib/aws — PASS

---

## Phase 5: Clean Up Core go.mod + App Integration — COMPLETE

- [x] Removed 29 cloud SDK deps from core `go.mod` via `go mod edit -droprequire`:
  - 7 `cloud.google.com/*` packages
  - 2 Azure packages (`Azure/azure-sdk-for-go/sdk/internal`, `AzureAD/microsoft-authentication-library-for-go`)
  - 12 AWS packages (`aws-sdk-go-v2/*`)
  - 3 `GoogleCloudPlatform/opentelemetry-operations-go/*`
  - 5 Google-only transitive deps (`google/s2a-go`, `googleapis/enterprise-certificate-proxy`, `googleapis/gax-go`, etc.)
- [x] Removed `hashicorp/errwrap`, `hashicorp/go-multierror` (golang-migrate transitive deps)
- [x] Removed `golang-jwt/jwt/v5`, `pkg/browser`, `kylelemons/godebug` (Azure-only transitive deps)
- [x] `go mod tidy -e` passes on core
- [x] Verify: core go.mod has 0 cloud SDK deps (was 21)
- [x] Verify: core go.mod went from ~105 lines to ~86 lines
- [x] All 4 app container.go files updated with `contrib/postgres` blank import
- [x] All 4 app go.mod files have require + replace for contrib/postgres
- [x] Verify: service-admin builds with tags `google_uuidv7,mock_auth,mock_storage,noop,vanilla,lyngua` — PASS
- [x] Verify: service-admin alpha build via `build-alpha.ps1` — PASS (34 MB binary)

### go.mod Before/After (core espyna)
| Metric | Before | After |
|--------|--------|-------|
| Direct deps | 14 | 14 |
| Indirect deps | ~85 | ~72 |
| Cloud SDK deps | 21 | 0 |
| Total lines | ~139 | ~86 |

### service-admin go.mod — Cloud SDKs Eliminated
- Before: Azure (4), AWS (14), Google Cloud (15+) = 33+ cloud SDK indirect deps
- After: 0 cloud SDK deps
- Remaining indirect noise: Gin (6 pkgs) + Fiber (6 pkgs) — deferred to Phase 6

---

## Phase 6: contrib/gin + contrib/fiber — COMPLETE

- [x] Explore Gin/Fiber adapter coupling to composition internals
- [x] Create `composition/` re-export packages (core, contracts, routing, routing/customization)
- [x] Create `contrib/gin/go.mod` and move Gin adapter + 6 middleware files
- [x] Create `contrib/fiber/go.mod` and move Fiber v2 + v3 adapters + 3 middleware files
- [x] Remove Gin/Fiber deps from core go.mod
- [x] Add `contrib/gin` and `contrib/fiber` to `go.work`
- [x] Verify: `go build ./...` in contrib/gin — PASS
- [x] Verify: `go build ./...` in contrib/fiber — PASS
- [x] Verify: service-admin builds clean — PASS

**Agent:** `http-framework-builder` — completed Phase 6 in previous session

### Composition Re-Export Packages Created
The key challenge was that Gin/Fiber adapters imported `internal/composition/*` types. Solved by creating 4 new re-export packages:

| Package | Re-exports | Key types |
|---|---|---|
| `composition/core/` | `internal/composition/core/` | `Container` |
| `composition/contracts/` | `internal/composition/contracts/` | `ProtobufParser`, `UseCaseHandler`, `Route`, `Request`, `Response`, `CORSConfig` |
| `composition/routing/` | `internal/composition/routing/` | `Route`, `RouteMetadata`, `RouteGroup`, `Config` |
| `composition/routing/customization/` | `internal/composition/routing/customization/` | `RouteCustomizer`, `CustomizationConfig`, `NewRouteCustomizer` |

### Files Created
**contrib/gin/** (10 files):
- `register.go`, `go.mod`, `go.sum`
- `internal/adapter/adapter.go` (329 lines — self-registers as "gin")
- `internal/adapter/middleware/` — authentication, authorization, business_type, cors, csrf, gzip

**contrib/fiber/** (8 files):
- `register.go`, `go.mod`, `go.sum`
- `internal/adapter/adapter.go` (Fiber v2)
- `internal/adapterv3/adapter.go` (Fiber v3)
- `internal/adapter/middleware/` — business_type, cors, gzip

---

## Summary

- **Phases complete:** 6 / 6 (ALL COMPLETE)
- **Contrib modules created:** 6 (postgres, google, azure, aws, gin, fiber)
- **Re-export packages created:** 11 (ports, registry, database/interfaces, database/model, database/operations, database/transactions, storage/helpers, composition/core, composition/contracts, composition/routing, composition/routing/customization)
- **Cloud/framework SDKs removed from core:** 29 cloud SDK + Gin/Fiber deps
- **Core go.mod:** 14 direct deps, ~72 indirect, 0 cloud/framework SDKs
- **Binary size:** 34 MB (unchanged — Go linker already eliminated dead code)
- **Build verified:** all 6 contrib modules + core + service-admin

---

## Skipped / Deferred

| Item | Reason |
|------|--------|
| `run.ps1` tag cleanup | `postgresql` tag is harmless (Go ignores unknown tags), other providers still use tags |
| E2E tests | Require running server — builds confirmed, functional testing deferred |
| Alpha deployment test | `.env.alpha` uses `gcp_storage` — app needs `import _ contrib/google` for production (dev uses mock_storage) |

---

## All Phases Complete

### Final state (2026-03-06)

All 6 phases are complete. The espyna-golang core module has zero cloud SDK and zero HTTP framework SDK dependencies. All heavy SDKs are isolated in 6 contrib sub-modules.

### To verify the split works:

```bash
# Build core espyna (should have 0 cloud/framework SDK deps)
cd packages/espyna-golang-ryta && go build ./...

# Build all 6 contrib modules
for mod in postgres google azure aws gin fiber; do
  cd packages/espyna-golang-ryta/contrib/$mod && go build ./...
done

# Build service-admin (dev mode)
cd apps/service-admin && go build -tags "google_uuidv7,mock_auth,mock_storage,noop,vanilla,lyngua" ./...

# Build service-admin (alpha mode — cross-compile Linux AMD64)
cd apps/service-admin && powershell -ExecutionPolicy Bypass -File scripts/build-alpha.ps1
```

### Known Issues:

1. **Alpha deployment**: `.env.alpha` sets `CONFIG_STORAGE_PROVIDER=gcp_storage` but service-admin only imports `contrib/postgres`. For alpha to work with GCS, add `import _ "github.com/erniealice/espyna-golang/contrib/google"` to container.go and update go.mod.
2. **Remaining consumer registration files** in core (kept — these are lightweight, no cloud SDKs):
   - `register_auth_jwt.go` (stdlib only)
   - `register_email_microsoft.go` (custom HTTP client)
   - `register_payment_*.go` (paypal, asiapay, maya — custom HTTP clients)
   - `register_scheduler_calendly.go` (custom HTTP client)
   - `register_server_vanilla.go` (stdlib HTTP adapter)
   - `register_translation_lyngua.go` (lyngua package)
