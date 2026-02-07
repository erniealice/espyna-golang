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
    → blank-imports real adapters → their init() self-registers
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

### Adapter Wrappers

Thin wrappers that expose internal provider functionality to consumer apps.
Each file provides a `NewXxxAdapterFromContainer(container)` constructor.

| File | Adapter | Key Methods |
|------|---------|-------------|
| `adapter_auth.go` | AuthAdapter | VerifyToken, CreateCustomToken |
| `adapter_database.go` | DatabaseAdapter | Create, Read, Update, Delete, List, Query |
| `adapter_email.go` | EmailAdapter | SendEmail, SendSimpleEmail, SendHTMLEmail |
| `adapter_id.go` | IDAdapter | GenerateID |
| `adapter_payment.go` | PaymentAdapter | CreatePayment, VerifyPayment, ProcessWebhook |
| `adapter_scheduler.go` | SchedulerAdapter | CreateSchedule, CancelSchedule, CheckAvailability |
| `adapter_server.go` | ServerAdapter | Start, RegisterRoute, RegisterMiddleware |
| `adapter_storage.go` | StorageAdapter | Upload, Download, Delete, GetSignedURL |

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

| File | Build Tag | Adapter | External Deps |
|------|-----------|---------|---------------|
| **HTTP Servers (pick one)** ||||
| `register_server_gin.go` | `gin` | Gin HTTP server | gin-gonic/gin |
| `register_server_fiber.go` | `fiber` | Fiber v2 | gofiber/fiber/v2 |
| `register_server_fiberv3.go` | `fiber_v3` | Fiber v3 | gofiber/fiber/v3 |
| `register_server_vanilla.go` | `vanilla` | net/http server | None (stdlib) |
| **Database (pick one)** ||||
| `register_database_firestore.go` | `firestore` | Google Firestore | cloud.google.com/go/firestore |
| `register_database_postgres.go` | `postgres` | PostgreSQL | github.com/lib/pq |
| **Auth (pick one)** ||||
| `register_auth_firebase.go` | `firebase` | Firebase Auth | firebase.google.com/go |
| `register_auth_jwt.go` | `jwt_auth` | JWT auth | (minimal) |
| **Email (pick one)** ||||
| `register_email_gmail.go` | `google && gmail` | Gmail API | Google API client |
| `register_email_microsoft.go` | `microsoft && microsoftgraph` | MS Graph email | MS Graph SDK |
| **Payment (can combine)** ||||
| `register_payment_asiapay.go` | `asiapay` | AsiaPay | None (net/http) |
| `register_payment_maya.go` | `maya` | Maya | None (net/http) |
| `register_payment_paypal.go` | `paypal` | PayPal | None (net/http) |
| **Scheduler (pick one)** ||||
| `register_scheduler_calendly.go` | `calendly` | Calendly | None (net/http) |
| **Storage (pick one)** ||||
| `register_storage_gcs.go` | `google && gcs` | Google Cloud Storage | cloud.google.com/go/storage |
| `register_storage_s3.go` | `aws && s3` | AWS S3 | AWS SDK v2 |
| `register_storage_azure.go` | `azure && azureblob` | Azure Blob | Azure SDK |
| **Tabular (pick one)** ||||
| `register_tabular_googlesheets.go` | `google && googlesheets` | Google Sheets | Google Sheets API |
| **Translation** ||||
| `register_translation_lyngua.go` | `lyngua` | Lyngua | leapfor.xyz/lyngua |

## Build Tag Examples

```bash
# Minimal (development) — mocks only, smallest binary
go build -tags "mock_db,mock_auth,mock_email,mock_payment,mock_storage,vanilla" ./...

# Staging — real DB, mock services
go build -tags "firestore,mock_auth,mock_email,mock_payment,mock_storage,gin,google" ./...

# Production: PH stack
go build -tags "firestore,firebase,gmail,maya,asiapay,calendly,gin,google,googlesheets" ./...

# Production: US stack
go build -tags "postgres,jwt_auth,gmail,paypal,gin,google,googlesheets,aws,s3" ./...
```

## Adding a New Adapter

Follow this recipe when you need to add a new adapter (e.g., Google Calendar
for the scheduler category).

### Step 1: Create the adapter package

```
internal/infrastructure/adapters/secondary/scheduler/googlecalendar/
    adapter.go
    types.go
    config.go
```

**CRITICAL: Every `.go` file MUST have the build tag.**

```go
//go:build googlecalendar

package googlecalendar
```

If even one file is missing the tag, its dependencies will be pulled into
ALL binaries regardless of build tags.

### Step 2: Implement self-registration

In `adapter.go`, add an `init()` function that registers with the global
factory registry:

```go
func init() {
    registry.RegisterSchedulerProvider("googlecalendar", newFactory, transformConfig)
    registry.RegisterSchedulerBuildFromEnv("googlecalendar", buildFromEnv)
}
```

The three things to register:
1. **Factory function** — creates an uninitialized adapter instance
2. **Config transformer** — converts raw config map to typed config
3. **BuildFromEnv function** — creates a fully-configured adapter from env vars

### Step 3: Create the register file

Create `consumer/register_scheduler_googlecalendar.go`:

```go
//go:build googlecalendar

package consumer

import _ "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/scheduler/googlecalendar"
```

That's it — 3 lines. The file name follows the convention: `register_{category}_{adapter}.go`.

### Step 4: Update this README

Add the new adapter to the build tag table above.

### Step 5: Use it

```bash
go build -tags "googlecalendar,firestore,gin,google" ./...
```

```env
CONFIG_SCHEDULER_PROVIDER=googlecalendar
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Consumer App (bfit-subs-golang-v2, be-master, etc.)        │
│                                                             │
│  import "leapfor.xyz/espyna/consumer"                       │
│                                                             │
│  container := consumer.NewContainerFromEnv()                │
│  container.Initialize()                                     │
│  db := consumer.NewDatabaseAdapterFromContainer(container)  │
└─────────────┬───────────────────────────────────────────────┘
              │ imports
              v
┌─────────────────────────────────────────────────────────────┐
│  consumer/ package                                          │
│                                                             │
│  consumer.go ──── API surface (Container, UseCases, etc.)   │
│  adapter_*.go ─── Thin wrappers over internal providers     │
│  register.go ──── Blank imports for mocks (always compiled) │
│  register_{cat}_{name}.go ── Blank imports gated by tags    │
│       │                                                     │
│       │ blank imports trigger init()                        │
│       v                                                     │
│  internal/infrastructure/adapters/                          │
│       │                                                     │
│       │ each adapter's init() calls:                        │
│       v                                                     │
│  internal/infrastructure/registry/                          │
│       │  FactoryRegistry[T, C] — generic type-safe map      │
│       │  RegisterXxxProvider("name", factory, config)        │
│       │  BuildXxxProviderFromEnv("name") → provider         │
│       │                                                     │
│       │ at runtime, composition layer reads env vars:        │
│       v                                                     │
│  internal/composition/providers/infrastructure/             │
│       CONFIG_DATABASE_PROVIDER=firestore                    │
│       → registry.BuildDatabaseProviderFromEnv("firestore")  │
│       → returns configured Firestore adapter                │
└─────────────────────────────────────────────────────────────┘
```

## FAQ

### Why are there separate register files instead of one big file?

**Binary safety.** If `register.go` imports a Firestore adapter package,
Go will enter that directory even when you build without `-tags firestore`.
If any `.go` file in that directory lacks a build tag, its dependencies get
pulled in — silently adding megabytes to your binary.

By giving each adapter its own build-tagged register file, Go never even
enters the adapter directory unless you opt in via the build tag.

### Why do mock adapters stay in register.go (always compiled)?

Mock adapters have zero external dependencies. Even if their source files
have build tags (like `mock_db`), the blank import is harmless — Go enters
the directory, finds no compilable files, and moves on. No binary impact.

### Can I compile multiple payment adapters?

Yes. Payment adapters can coexist. Build with `-tags "maya,asiapay,paypal"`
and all three register. The `CONFIG_PAYMENT_PROVIDER` env var picks which
one is active at runtime.

### Can I compile multiple HTTP servers?

No. HTTP server adapters use mutual exclusion tags (e.g., fiber's tag
includes `!gin && !fiber_v3`). Pick exactly one.

### What env vars control adapter selection?

| Variable | Values | Default |
|----------|--------|---------|
| `CONFIG_DATABASE_PROVIDER` | mock_db, postgres, firestore | mock_db |
| `CONFIG_AUTH_PROVIDER` | mock_auth, firebase_auth | mock_auth |
| `CONFIG_EMAIL_PROVIDER` | mock_email, gmail, microsoft | mock_email |
| `CONFIG_PAYMENT_PROVIDER` | mock_payment, asiapay, maya, paypal | mock_payment |
| `CONFIG_SCHEDULER_PROVIDER` | mock, calendly | mock |
| `CONFIG_STORAGE_PROVIDER` | mock_storage, local, gcs, s3, azure | mock_storage |
| `CONFIG_ID_PROVIDER` | noop, google_uuidv7 | noop |
| `CONFIG_TABULAR_PROVIDER` | mock, googlesheets | mock |
| `CONFIG_TRANSLATION_PROVIDER` | noop, file, mock, lyngua | noop |
