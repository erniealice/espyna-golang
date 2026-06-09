# espyna/consumer

The `consumer` package is the **public API surface** of the espyna framework.
External applications import this single package to get access to the
container, adapter wrappers, and use cases. Adapter registration is handled
automatically via build-tagged blank imports.

## How It Works

```
Your app imports espyna/consumer
    |
    v
consumer/register.go (always compiled)
    → blank-imports mock/noop adapters → their init() self-registers
    |
consumer/register_{category}_{adapter}.go (compiled only when build tag is set)
    → blank-imports real adapters (from contrib/ or secondary/) → their init() self-registers
    |
    v
At runtime, CONFIG_*_PROVIDER env vars select which registered adapter to use
```

Build tags control what goes into your binary. Only adapters whose tags
match your build command are compiled in. Mock adapters are always available
as fallbacks.

## Directory Map

### Core API Surface

| File | Purpose |
|------|---------|
| `consumer.go` | Container, NewContainer, NewContainerFromEnv, RouteExporter, type aliases, re-exported options |
| `context.go` | WithUserID, ExtractUserIDFromContext — context utilities |
| `server_gin.go` | CreateGinHandler, InstallRouteOnGin — Gin-specific helpers (//go:build gin) |

### Adapter Wrappers (driven-port facades)

Thin substitution-driven wrappers that expose internal provider functionality
to consumer apps. Each file provides a `NewXxxAdapterFromContainer(container)`
constructor. Inventory post-2026-05-18 + planned additions:

| File | Adapter | Key Methods | Status |
|------|---------|-------------|--------|
| `adapter_auth.go` | AuthAdapter | VerifyToken, CreateCustomToken | Live |
| `adapter_database.go` | DatabaseAdapter | Create, Read, Update, Delete, List, Query | Live |
| `adapter_email.go` | EmailAdapter | SendEmail, SendSimpleEmail, SendHTMLEmail | Live |
| `adapter_server.go` | ServerAdapter | Start, RegisterRoute, RegisterMiddleware | Live |
| `adapter_storage.go` | StorageAdapter | Upload, Download, Delete, GetSignedURL | Live |
| `adapter_email.go` (EmailRouter) | EmailRouter | Route across plural providers | Planned (N3 P3) |
| `adapter_auth.go` (AuthRouter) | AuthRouter | Route login/callback across providers | Planned (E1 P1) |
| `adapter_storage.go` (StorageRouter) | StorageRouter | Route ops by repository provider | Planned (E3 P1) |

### Admission policy (Q6, locked 2026-05-18)

A new `consumer/adapter_*.go` file is admitted only if ONE of these tests holds:

- **(A)** ≥2 real implementations exist AND ≥1 actual caller exists in the monorepo.
- **(B)** The file wraps a port under `internal/application/ports/` AND has ≥1 actual caller AND the port enables tests (mock impl exists/roadmapped) or cross-package compilation visibility.
- **(C) — REJECTED:** "exported public API for downstream consumers without callers" is explicitly NOT a valid admission rationale. Speculative facades fail.

The PR description must cite which test is satisfied for every new file.

### Previously deleted adapter facades

Eight `consumer/adapter_*.go` files were removed in 2026-05-18:

| File | Reason | Re-promotion status |
|------|--------|---------------------|
| `adapter_fulfillment.go` | Tier 3 — 0 callers | **Re-promote when N5 (fulfillment build-out) lands** — integration use cases will be the caller |
| `adapter_payment.go` | Tier 3 — 0 callers | **Re-promote when E2 (multi-provider payments) lands** — webhook route dispatch will be the caller |
| `adapter_scheduler.go` | Tier 3 — 0 callers | Re-promote when needed |
| `adapter_id.go` | Tier 3 — 0 callers | Not needed — ID wired via DI only |
| `adapter_treasury_advances.go` | USE_CASE_WRAPPER | Not re-promoting — callers go through `uc.Treasury.*` directly |
| `adapter_session.go` | USE_CASE_WRAPPER | Not re-promoting — service-admin calls `uc.Auth.*` directly |
| `adapter_audit.go` | Tier 2 visibility-bridge | Not re-promoting — audit via service use case |
| `adapter_ledger.go` | Tier 2 visibility-bridge | Not re-promoting — reporting via service use cases |

### Registration Files

These files contain **only blank imports** that trigger adapter `init()`
functions. Each adapter self-registers with the global factory registry.

File naming convention: `register_{category}_{adapter}.go`

#### Always compiled (register.go)

Mock adapters, noop defaults, and lightweight stdlib-only adapters.
These have zero external dependencies and add negligible binary size.

#### Build-tag gated (register_{category}_{adapter}.go)

Each real adapter gets its own file with a matching `//go:build` tag.
The adapter is only compiled into the binary when the tag is present.

> **Canonical naming (in-flight 2026-06-09):** every build tag, registry key,
> `Name()` return, and `CONFIG_*_PROVIDER` value uses ONE canonical token per
> provider. No aliases. See [provider-system wiki](../../docs/wiki/articles/provider-system.md).

| File | Build Tag | Adapter | Location | External Deps |
|------|-----------|---------|----------|---------------|
| **HTTP Servers (pick one)** |||||
| `register_server_http.go` | `http` | stdlib net/http | `contrib/http` | None (stdlib) |
| `register_server_gin.go` | `gin` | Gin HTTP server | `contrib/gin` | gin-gonic/gin |
| `register_server_fiber.go` | `fiber` | Fiber v2/v3 | `contrib/fiber` | gofiber/fiber |
| **Database (pick one)** |||||
| `register_database_postgres.go` | `postgresql` | PostgreSQL | `contrib/postgres` | github.com/lib/pq |
| `register_database_firestore.go` | `firestore` | Firestore | `contrib/google` | cloud.google.com/go/firestore |
| **Auth (pick one or combine)** |||||
| `register_auth_firebase.go` | `firebase_auth` | Firebase Auth | `contrib/google` | firebase.google.com/go |
| **Email (pick one or combine)** |||||
| `register_email_google.go` | `google_email` | Gmail API | `contrib/google` | Google API client |
| `register_email_microsoft.go` | `microsoft_email` | MS Graph email | `contrib/microsoft` | MS Graph SDK |
| **Payment (can combine)** |||||
| `register_payment_asiapay.go` | `asiapay` | AsiaPay | `contrib/asiapay` | None (net/http) |
| `register_payment_maya.go` | `maya` | Maya | `contrib/maya` | None (net/http) |
| `register_payment_paypal.go` | `paypal` | PayPal | `contrib/paypal` | None (net/http) |
| **Scheduler (can combine)** |||||
| `register_scheduler_calendly.go` | `calendly` | Calendly | `contrib/calendly` | None (net/http) |
| `register_scheduler_google_calendar.go` | `google_calendar` | Google Calendar | `contrib/google` | Google Calendar API |
| **Fulfillment (can combine)** |||||
| `register_fulfillment_lalamove.go` | `lalamove` | Lalamove | `contrib/lalamove` | None (net/http) |
| `register_fulfillment_grabexpress.go` | `grabexpress` | GrabExpress | `contrib/grabexpress` | None (net/http) |
| **Storage (pick one or combine)** |||||
| `register_storage_gcp.go` | `gcp_storage` | Google Cloud Storage | `contrib/google` | cloud.google.com/go/storage |
| `register_storage_aws.go` | `aws_storage` | AWS S3 | `contrib/aws` | AWS SDK v2 |
| `register_storage_azure.go` | `azure_storage` | Azure Blob | `contrib/azure` | Azure SDK |
| `register_storage_sharepoint.go` | `sharepoint_storage` | SharePoint | `contrib/microsoft` | MS Graph SDK |
| **Tabular** |||||
| `register_tabular_google_sheets.go` | `google_sheets` | Google Sheets | `contrib/google` | Google Sheets API |
| **ID** |||||
| `register_id_uuidv7.go` | `google_uuidv7` | UUIDv7 (Google) | `contrib/google` | github.com/google/uuid |

## Build Tag Examples

```bash
# Minimal (development) — mocks only, smallest binary
go build -tags "http,mock_db,mock_auth,mock_storage,mock_email" ./cmd/server

# Standard dev with real DB
go build -tags "http,postgresql,mock_auth,mock_email" ./cmd/server

# Production GCP stack
go build -tags "http,postgresql,firebase_auth,gcp_storage,google_uuidv7,google_email,asiapay,calendly" ./cmd/server

# Production with multiple payment gateways
go build -tags "http,postgresql,firebase_auth,gcp_storage,google_uuidv7,google_email,maya,asiapay,paypal" ./cmd/server
```

## Adding a New Adapter

Follow this recipe when adding a new adapter.

### Step 0: Decide placement

| Adapter type | Location | Module |
|-------------|----------|--------|
| Vendor API (Lalamove, Calendly, etc.) | `contrib/{vendor}/internal/adapter/` | New `contrib/{vendor}/go.mod` |
| Vendor API, vendor already in contrib/ (Gmail → Google) | `contrib/{vendor}/internal/{type}/{adapter}/` | Existing `contrib/{vendor}/go.mod` |
| Mock / noop / stdlib-only | `secondary/{type}/{adapter}/` | Main espyna module |

### Step 1: Create the adapter package

**For `contrib/` (new module):**
```
contrib/myprovider/
    go.mod                          ← separate Go module
    register.go                     ← empty package decl (always compiles)
    register_myprovider.go          ← //go:build myprovider
    internal/adapter/
        adapter.go                  ← //go:build myprovider — the implementation
        stub.go                     ← empty package decl (no tag, so import resolves)
```

**For `contrib/` (existing module, e.g., contrib/google):**
```
contrib/google/
    register_myprovider.go          ← //go:build myprovider (NEW)
    internal/{type}/myprovider/
        adapter.go                  ← //go:build myprovider — the implementation (NEW)
        stub.go                     ← empty package decl (NEW)
```

**For `secondary/` (mock/noop):**
```
internal/infrastructure/adapters/secondary/{type}/{adapter}/
    adapter.go                      ← the implementation
    stub.go                         ← //go:build !{real_tag} if mutual exclusion needed
```

### Step 2: Implement self-registration

Use ONE canonical token everywhere:

```go
//go:build myprovider

package myprovider

func init() {
    registry.Register{Type}Provider("{canonical_token}", newFactory, transformConfig)
    registry.Register{Type}BuildFromEnv("{canonical_token}", buildFromEnv)
}

func (p *MyAdapter) Name() string {
    return "{canonical_token}"  // SAME token as registry key
}
```

### Step 3: Create the consumer register file

Create `consumer/register_{category}_{adapter}.go`:

```go
//go:build myprovider

package consumer

import _ "github.com/erniealice/espyna-golang/contrib/myprovider"
```

### Step 4: Add to audit-tags.sh

Add a mode in `apps/service-admin/scripts/audit-tags.sh` that verifies your
adapter compiles in and competing adapters do not.

### Step 5: Update this README + wiki

Add the new adapter to the build-tag table above and update
[provider-system wiki article](../../docs/wiki/articles/provider-system.md).

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Consumer App (service-admin, etc.)                          │
│                                                             │
│  import "espyna/consumer"                                    │
│                                                             │
│  container := consumer.NewContainerFromEnv()                │
│  container.Initialize()                                     │
└─────────────┬───────────────────────────────────────────────┘
              │ imports
              v
┌─────────────────────────────────────────────────────────────┐
│  consumer/ package                                          │
│                                                             │
│  consumer.go ──── API surface (Container, UseCases, etc.)   │
│  adapter_*.go ─── Thin wrappers / routers over providers    │
│  register.go ──── Blank imports for mocks (always compiled) │
│  register_{cat}_{name}.go ── Blank imports gated by tags    │
│       │                                                     │
│       │ blank imports trigger init()                        │
│       v                                                     │
│  TWO adapter locations:                                     │
│                                                             │
│  internal/infrastructure/adapters/secondary/                │
│       └── mock, noop, pure-stdlib adapters                  │
│                                                             │
│  contrib/{vendor}/internal/{type}/{adapter}/                │
│       └── vendor SDK adapters (separate go.mod per vendor)  │
│           shared code in contrib/{vendor}/internal/common/  │
│       │                                                     │
│       │ each adapter's init() calls:                        │
│       v                                                     │
│  internal/infrastructure/registry/                          │
│       FactoryRegistry[T, C] — generic type-safe map         │
│       Register{Type}BuildFromEnv("{token}", BuildFromEnv)   │
│       Build{Type}ProviderFromEnv("{token}") → provider      │
│       │                                                     │
│       │ at runtime, composition reads env vars:             │
│       v                                                     │
│  internal/composition/providers/{infrastructure,integration}│
│       CONFIG_*_PROVIDER → registry.Build*FromEnv(token)     │
│       → returns the configured adapter                      │
└─────────────────────────────────────────────────────────────┘
```

## FAQ

### Why are there separate register files instead of one big file?

**Binary safety.** Each adapter's register file is build-tagged. Go never
enters the adapter directory unless you opt in via the build tag.

### Why do mock adapters stay in register.go (always compiled)?

Mock adapters have zero external dependencies and negligible binary impact.

### Can I compile multiple payment/fulfillment adapters?

Yes. Plural provider types (payment, scheduler, fulfillment) support
simultaneous adapters. Build with multiple tags and set `CONFIG_*_PROVIDER`
to a comma-separated list.

### Can I compile multiple HTTP servers?

No. HTTP server adapters use mutual exclusion (audit-tags.sh enforces it).

### What env vars control adapter selection?

| Variable | Canonical Tokens | Default |
|----------|-----------------|---------|
| `CONFIG_DATABASE_PROVIDER` | `postgresql`, `firestore`, `mock_db` | `mock_db` |
| `CONFIG_AUTH_PROVIDER` | `password`, `firebase`, `mock` | `mock` |
| `CONFIG_EMAIL_PROVIDER` | `google_email`, `microsoft_email`, `mock_email` | `mock_email` |
| `CONFIG_PAYMENT_PROVIDER` | `maya`, `asiapay`, `paypal`, `xero`, `mock_payment` | `mock_payment` |
| `CONFIG_SCHEDULER_PROVIDER` | `calendly`, `google_calendar`, `mock_scheduler` | `mock_scheduler` |
| `CONFIG_FULFILLMENT_PROVIDER` | `lalamove`, `grabexpress`, `mock_fulfillment` | `mock_fulfillment` |
| `CONFIG_STORAGE_PROVIDER` | `gcp_storage`, `aws_storage`, `azure_storage`, `local_storage`, `mock_storage` | `mock_storage` |
| `CONFIG_ID_PROVIDER` | `google_uuidv7`, `noop` | `noop` |
| `CONFIG_SERVER_PROVIDER` | `http`, `gin`, `fiber`, `grpc` | `http` |
| `CONFIG_TABULAR_PROVIDER` | `google_sheets`, `mock_tabular` | `mock_tabular` |
