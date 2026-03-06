# Espyna

Pluggable Go backend framework built on hexagonal architecture. Provides use cases, DI container, adapter registry, and HTTP server abstraction for business management APIs.

## Architecture

```
                         ┌─────────────────────────────────────────────┐
                         │              Consumer App                   │
                         │  import _ "contrib/postgres"                │
                         │  import _ "contrib/gin"                     │
                         │  container := consumer.NewContainerFromEnv()│
                         └──────────────┬──────────────────────────────┘
                                        │
                    ┌───────────────────┐│┌───────────────────┐
                    │   contrib/postgres│││   contrib/gin     │
                    │   contrib/google  │││   contrib/fiber   │
                    │   contrib/azure   │││                   │
                    │   contrib/aws     │││                   │
                    └───────┬───────────┘│└───────┬───────────┘
                            │            │        │
                    ┌───────▼────────────▼────────▼───────────┐
                    │           espyna-golang (core)           │
                    │                                          │
                    │  consumer/     ← public entry point      │
                    │  ports/        ← interface contracts      │
                    │  registry/     ← factory registry         │
                    │  composition/  ← re-exported DI types     │
                    │  database/     ← re-exported DB types     │
                    │  storage/      ← re-exported storage      │
                    │                                          │
                    │  internal/application/usecases/           │
                    │  internal/composition/                    │
                    │  internal/infrastructure/adapters/        │
                    │  internal/infrastructure/registry/        │
                    └──────────────────────────────────────────┘
```

### How it works

1. **Ports** define interfaces — use cases depend only on these (never on concrete adapters)
2. **Adapters** implement ports — each registers itself via `init()` into the central registry
3. **Container** reads env config, looks up registered adapters by name, wires everything
4. **Consumer app** selects adapters by importing contrib modules (blank import triggers `init()`)

## Module Structure

Espyna is a Go multi-module project. The core module has zero cloud SDK dependencies. Heavy SDKs are isolated in `contrib/` sub-modules, each with its own `go.mod`.

### Core (`github.com/erniealice/espyna-golang`)

| Directory | What it does |
|---|---|
| `consumer/` | Public API — `NewContainerFromEnv()`, `ServerAdapter`, use case access |
| `internal/application/usecases/` | Business logic — entity, product, revenue, inventory, subscription, asset, treasury, ledger, workflow |
| `internal/application/ports/` | All port interfaces (repositories, services) |
| `internal/composition/` | DI container, provider initialization, route management |
| `internal/infrastructure/adapters/` | Built-in adapters (mock, noop, vanilla HTTP, JWT, local storage, payment gateways) |
| `internal/infrastructure/registry/` | Generic `FactoryRegistry[T, C]` — heart of adapter discovery |
| `internal/orchestration/` | Workflow engine |

### Re-export Packages

Contrib sub-modules (separate `go.mod`) cannot import `internal/` across module boundaries. These public packages re-export internal types via `type X = internal.X`:

| Public path | Re-exports from | Key types |
|---|---|---|
| `ports/` | `internal/application/ports/` | 60+ port interfaces, entity constants, error constructors |
| `registry/` | `internal/infrastructure/registry/` | `FactoryRegistry[T,C]`, `RegisterDatabaseProvider`, `RegisterServerProvider` |
| `database/interfaces/` | `internal/.../database/common/interface/` | `DatabaseOperation`, `TransactionAware`, `QueryBuilder` |
| `database/model/` | `internal/.../database/common/model/` | `BaseModel`, `TransactionError`, `DatabaseError` |
| `database/operations/` | `internal/.../database/common/operations/` | `ConvertMapToProtobuf`, `ConvertSliceToProtobuf` |
| `database/transactions/` | `internal/.../database/common/transactions/` | `TransactionServiceAdapter` |
| `storage/helpers/` | `internal/.../storage/common/` | `GenerateObjectID`, `DetectContentType` |
| `composition/core/` | `internal/composition/core/` | `Container` |
| `composition/contracts/` | `internal/composition/contracts/` | `ProtobufParser`, `UseCaseHandler`, `Route`, `Request`, `Response` |
| `composition/routing/` | `internal/composition/routing/` | `Route`, `RouteMetadata`, `RouteGroup` |
| `composition/routing/customization/` | `internal/composition/routing/customization/` | `RouteCustomizer`, `NewRouteCustomizer` |

### Contrib Sub-Modules

Each is a separate Go module with its own `go.mod`. Apps opt in by blank-importing.

| Module | SDK deps isolated | Adapter type |
|---|---|---|
| `contrib/postgres` | `lib/pq`, `golang-migrate` | Database (68 adapters), migration, ledger, reference checker |
| `contrib/google` | `cloud.google.com/go/*`, `firebase.google.com/*` | Firebase auth, Firestore DB, GCS storage, Gmail, Google Sheets |
| `contrib/azure` | `github.com/Azure/azure-sdk-for-go/*` | Azure Blob Storage |
| `contrib/aws` | `github.com/aws/aws-sdk-go-v2/*` | AWS S3 Storage |
| `contrib/gin` | `github.com/gin-gonic/gin`, `gin-contrib/*` | Gin HTTP server + middleware |
| `contrib/fiber` | `github.com/gofiber/fiber/v2`, `v3` | Fiber v2/v3 HTTP server + middleware |

## Self-Registration Pattern

Every adapter registers itself in `init()`. No hardcoded imports in core.

```go
// contrib/postgres/internal/adapter/adapter.go
func init() {
    registry.RegisterDatabaseProvider(
        "postgresql",
        func() ports.DatabaseProvider { return NewPostgresAdapter() },
        transformConfig,
    )
}
```

Consumer apps activate adapters with blank imports:

```go
// apps/service-admin/internal/composition/container.go
import (
    "github.com/erniealice/espyna-golang/consumer"
    _ "github.com/erniealice/espyna-golang/contrib/postgres"
    // vanilla HTTP, mock_auth, mock_storage stay in core — no extra import needed
)

func main() {
    container := consumer.NewContainerFromEnv()
    server := consumer.NewServerAdapterFromContainer(container)
    server.Start(":8080")
}
```

## Built-in Adapters (in core, zero external SDKs)

These stay in core because they use only stdlib or lightweight deps:

| Category | Adapters | Notes |
|---|---|---|
| HTTP Server | vanilla | `net/http` stdlib, build tag: `vanilla` |
| Auth | jwt, mock, noop, database | JWT uses stdlib crypto |
| Storage | local, mock | Filesystem / in-memory |
| Database | mock | In-memory mock |
| Email | microsoft, mock | Custom HTTP client |
| Payment | paypal, asiapay, maya, mock | Custom HTTP clients |
| Scheduler | calendly, mock | Custom HTTP client |
| Translation | lyngua, file, mock, noop | Lightweight |
| ID | uuidv7, noop | `google/uuid` is tiny |
| Tabular | mock | In-memory |

## Build Tags

Some core adapters still use build tags (compile-time selection):

| Tag | What it enables |
|---|---|
| `vanilla` | Vanilla HTTP server adapter |
| `google_uuidv7` | UUIDv7 ID provider |
| `mock_auth` | Mock authentication |
| `mock_storage` | Mock storage |
| `noop` | No-op adapters (ID, etc.) |
| `lyngua` | Lyngua translation integration |

Contrib modules do **not** use build tags — importing them is the opt-in mechanism.

## Domain Use Cases

| Domain | Entities |
|---|---|
| `entity` | Client, user, role, permission, location, workspace, group, delegate, staff, admin |
| `product` | Product, product category |
| `revenue` | Revenue (sales/bookings) |
| `inventory` | Inventory |
| `subscription` | Subscription, subscription plan |
| `expenditure` | Expense |
| `treasury` | Collection, disbursement |
| `asset` | Asset, asset category |
| `ledger` | Ledger reporting |
| `workflow` | Workflow engine, templates |
| `event` | Event |
| `common` | Shared attribute operations |

## Development Setup

### go.work (monorepo)

```
go 1.25.1

use (
    ./apps/service-admin
    ./packages/espyna-golang-ryta
    ./packages/espyna-golang-ryta/contrib/postgres
    ./packages/espyna-golang-ryta/contrib/google
    ./packages/espyna-golang-ryta/contrib/azure
    ./packages/espyna-golang-ryta/contrib/aws
    ./packages/espyna-golang-ryta/contrib/gin
    ./packages/espyna-golang-ryta/contrib/fiber
    // ... other apps and packages
)
```

### Build & verify

```bash
# Core (should have 0 cloud SDK deps)
cd packages/espyna-golang-ryta && go build ./...

# Each contrib module
cd packages/espyna-golang-ryta/contrib/postgres && go build ./...
cd packages/espyna-golang-ryta/contrib/google && go build ./...
cd packages/espyna-golang-ryta/contrib/azure && go build ./...
cd packages/espyna-golang-ryta/contrib/aws && go build ./...
cd packages/espyna-golang-ryta/contrib/gin && go build ./...
cd packages/espyna-golang-ryta/contrib/fiber && go build ./...

# Consumer app (dev mode)
cd apps/service-admin && go build -tags "google_uuidv7,mock_auth,mock_storage,noop,vanilla,lyngua" ./...
```

## Key Dependencies

### Core

| Package | Purpose |
|---|---|
| `github.com/erniealice/esqyma` | Protobuf schemas (domain entities, infrastructure, integration) |
| `github.com/erniealice/lyngua` | Translation/i18n |
| `github.com/google/cel-go` | Common Expression Language (authorization rules) |
| `github.com/google/uuid` | UUID generation |
| `google.golang.org/protobuf` | Protobuf runtime |
| `google.golang.org/grpc` | gRPC runtime |
| `leapfor.xyz/copya` | Shared mock data |
| `leapfor.xyz/vya` | Shared utilities |
