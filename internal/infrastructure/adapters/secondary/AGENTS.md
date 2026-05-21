# secondary/

**Outbound Adapters** (Driven Adapters) - implement how the application talks to external systems. These are concrete implementations of ports (interfaces) defined in the application layer.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          APPLICATION LAYER                                   │
│                       application/ports/*.go                                 │
│        (DatabaseProvider, AuthProvider, EmailProvider, etc.)                │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │ implements
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SECONDARY ADAPTERS                                   │
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │  database/  │  │    auth/    │  │  storage/   │  │   email/    │       │
│  │             │  │             │  │             │  │             │       │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │       │
│  │ │firestore│ │  │ │firebase │ │  │ │   gcs   │ │  │ │  gmail  │ │       │
│  │ │postgres │ │  │ │   jwt   │ │  │ │   s3    │ │  │ │microsoft│ │       │
│  │ │  mock   │ │  │ │  mock   │ │  │ │  azure  │ │  │ │  mock   │ │       │
│  │ └─────────┘ │  │ │  noop   │ │  │ │  local  │ │  │ └─────────┘ │       │
│  └─────────────┘  │ └─────────┘ │  │ │  mock   │ │  └─────────────┘       │
│                   └─────────────┘  │ └─────────┘ │                         │
│                                    └─────────────┘                         │
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │  payment/   │  │     id/     │  │translation/ │  │   common/   │       │
│  │             │  │             │  │             │  │             │       │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │       │
│  │ │  maya   │ │  │ │ uuidv7  │ │  │ │  file   │ │  │ │firebase │ │       │
│  │ │asiapay  │ │  │ │  noop   │ │  │ │ lyngua  │ │  │ │   gcp   │ │       │
│  │ │  mock   │ │  │ └─────────┘ │  │ │  noop   │ │  │ │ google  │ │       │
│  │ └─────────┘ │  └─────────────┘  │ │  mock   │ │  │ │microsoft│ │       │
│  └─────────────┘                   │ └─────────┘ │  │ └─────────┘ │       │
│                                    └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EXTERNAL SYSTEMS                                     │
│   (Firestore, PostgreSQL, Firebase Auth, GCS, S3, Gmail, Maya, etc.)        │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
secondary/
├── auth/                       # Authentication & Authorization
│   ├── firebase/adapter.go     # Firebase Auth
│   ├── jwt/adapter.go          # JWT token handling
│   ├── mock/adapter.go         # Mock for testing
│   └── noop/adapter.go         # No-op (disabled auth)
│
├── database/                   # Data Persistence (40 repositories)
│   ├── common/                 # Shared interfaces & operations
│   │   ├── interface/          # Repository interfaces
│   │   ├── model/              # Shared models
│   │   ├── operations/         # CRUD operations interface
│   │   └── transactions/       # Transaction support
│   ├── firestore/              # Firestore implementation
│   │   ├── core/operations.go  # Firestore CRUD operations
│   │   ├── entity/             # 16 entity repositories
│   │   ├── event/              # 2 event repositories
│   │   ├── payment/            # 3 payment repositories
│   │   ├── product/            # 8 product repositories
│   │   ├── subscription/       # 6 subscription repositories
│   │   └── workflow/           # 6 workflow repositories
│   ├── mock/                   # In-memory (same domain structure)
│   └── postgres/               # PostgreSQL (same domain structure)
│
├── email/                      # Email Services
│   ├── gmail/adapter.go        # Google Gmail API
│   ├── microsoft/adapter.go    # Microsoft Graph API
│   └── mock/adapter.go         # Mock for testing
│
├── storage/                    # File/Blob Storage
│   ├── azure/adapter.go        # Azure Blob Storage
│   ├── gcs/adapter.go          # Google Cloud Storage
│   ├── s3/adapter.go           # AWS S3
│   ├── local/adapter.go        # Local filesystem
│   ├── mock/adapter.go         # In-memory mock
│   └── common/helpers.go       # GenerateObjectID, DetectContentType
│
├── payment/                    # Payment Gateways
│   ├── maya/adapter.go         # Maya (Philippines)
│   ├── asiapay/adapter.go      # AsiaPay
│   └── mock/adapter.go         # Mock for testing
│
├── id/                         # ID Generation
│   ├── uuidv7/adapter.go       # UUIDv7 generator
│   └── noop/adapter.go         # No-op (passthrough)
│
├── translation/                # i18n Services
│   ├── file/adapter.go         # File-based translations
│   ├── lyngua/adapter.go       # Lyngua service
│   ├── noop/adapter.go         # No-op (passthrough)
│   └── mock/adapter.go         # Mock for testing
│
└── common/                     # Shared Cloud Client Utilities
    ├── firebase/               # Firebase client setup
    ├── gcp/                    # GCP common utilities
    ├── google/                 # Google API clients
    └── microsoft/              # Microsoft Graph clients
```

## Adapter Pattern

Each adapter follows the same pattern:

```go
// 1. Package per provider
package firestore

// 2. Struct implementing port interface
type FirestoreDatabaseProvider struct {
    client *firestore.Client
    config *dbpb.DatabaseProviderConfig
}

// 3. Constructor
func NewAdapter() *FirestoreDatabaseProvider {
    return &FirestoreDatabaseProvider{}
}

// 4. Port interface implementation
func (p *FirestoreDatabaseProvider) Initialize(config *dbpb.DatabaseProviderConfig) error
func (p *FirestoreDatabaseProvider) Name() string
func (p *FirestoreDatabaseProvider) IsEnabled() bool
func (p *FirestoreDatabaseProvider) IsHealthy(ctx context.Context) error
func (p *FirestoreDatabaseProvider) Close() error

// 5. Self-registration in init()
func init() {
    registry.RegisterDatabaseBuildFromEnv("firestore", BuildFromEnv)
}
```

## Repository Self-Registration

Database repositories register themselves via `init()`:

```go
// In database/firestore/entity/client.go
func init() {
    registry.RegisterRepositoryFactory("firestore", "client",
        func(conn any, tableName string) (any, error) {
            ops := conn.(*FirestoreOperations)
            return NewClientRepository(ops, tableName), nil
        })
}
```

## Provider Types Summary

| Type | Port Interface | Implementations |
|------|----------------|-----------------|
| database | `ports.DatabaseProvider` | firestore, postgres, mock |
| auth | `ports.AuthProvider` | firebase, jwt, mock, noop |
| storage | `ports.StorageProvider` | gcs, s3, azure, local, mock |
| email | `ports.EmailProvider` | gmail, microsoft, mock |
| payment | `ports.PaymentProvider` | maya, asiapay, mock |
| id | `ports.IDGenerator` | uuidv7, noop |
| translation | `ports.Translator` | file, lyngua, noop, mock |

## Database Domain Structure (40 Repositories)

| Domain | Repositories | Count |
|--------|--------------|-------|
| entity | admin, client, client_attribute, delegate, delegate_client, group, location, location_attribute, manager, permission, role, role_permission, staff, user, workspace, workspace_user, workspace_user_role | 17 |
| event | event, event_client | 2 |
| payment | payment, payment_method, payment_profile | 3 |
| product | collection, collection_plan, price_product, product, product_attribute, product_collection, product_plan, resource | 8 |
| subscription | balance, invoice, plan, plan_settings, price_plan, subscription | 6 |
| workflow | activity, activity_template, stage, stage_template, workflow, workflow_template | 6 |

## Import Patterns

```go
// Specific adapter (rarely needed - use registry instead)
import "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/firestore"

// Common utilities
import "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/storage/common"
import "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"

// Prefer: Use registry for provider access
import "leapfor.xyz/espyna/internal/infrastructure/registry"
provider, _ := registry.BuildDatabaseProviderFromEnv("firestore")
```

## Adding a New Adapter

1. **Create folder**: `secondary/{type}/{provider}/`
2. **Create adapter.go**:
   ```go
   package myprovider

   type MyAdapter struct { ... }

   func NewAdapter() *MyAdapter { ... }
   func (a *MyAdapter) Initialize(config *pb.Config) error { ... }
   func (a *MyAdapter) Name() string { return "myprovider" }
   // ... implement full port interface

   func init() {
       registry.Register{Type}BuildFromEnv("myprovider", BuildFromEnv)
   }
   ```
3. **Add build tag** (if needed): `//go:build myprovider`
4. **Import in consumer** to trigger `init()` registration

## Related Packages

| Package | Purpose |
|---------|---------|
| `application/ports/` | Port interfaces adapters implement |
| `infrastructure/registry/` | Factory & instance registries |
| `composition/providers/` | Provider creation & management |

## Key Design Decisions

1. **Self-registration via init()** - Adapters register with registry, no central wiring
2. **One folder per provider** - Clear separation: `storage/gcs/`, `storage/s3/`
3. **Consistent adapter.go naming** - Entry point always named `adapter.go`
4. **Common utilities extracted** - Shared code in `{type}/common/` folders
5. **Domain-organized repositories** - Database repos grouped by business domain
6. **Build tags for optional deps** - Heavy cloud SDKs excluded unless needed
