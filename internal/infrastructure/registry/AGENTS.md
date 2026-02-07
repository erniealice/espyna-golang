# registry/

The **Provider Registry** - enables self-registration of infrastructure adapters via `init()` functions, eliminating hardcoded switch statements and enabling runtime provider selection.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ADAPTER IMPLEMENTATIONS                              │
│              (firestore, postgres, mock, maya, gmail, etc.)                 │
│                                                                             │
│   Each adapter calls registry.Register*() in its init() function            │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │ self-registration
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              REGISTRY PACKAGE                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    FactoryRegistry[T, C]                             │   │
│  │              Generic, type-safe factory storage                      │   │
│  │         (registry.go - shared by all provider types)                 │   │
│  └───────────────────────────────────┬─────────────────────────────────┘   │
│                                      │                                      │
│         ┌────────────┬───────────────┼───────────────┬────────────┐        │
│         ▼            ▼               ▼               ▼            ▼        │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ ┌──────────┐ ┌──────────┐     │
│  │database  │ │  auth    │ │   storage    │ │  email   │ │ payment  │     │
│  │  .go     │ │  .go     │ │    .go       │ │  .go     │ │  .go     │     │
│  └──────────┘ └──────────┘ └──────────────┘ └──────────┘ └──────────┘     │
│         │            │               │               │            │        │
│         └────────────┴───────────────┼───────────────┴────────────┘        │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         instances.go                                 │   │
│  │              Runtime instance management & health checks             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           COMPOSITION LAYER                                  │
│                    composition/providers/infrastructure/                     │
│              Calls registry.BuildFromEnv() or registry.GetFactory()         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Two Registry Types

### 1. FactoryRegistry (Compile-time)
Stores **factory functions** for creating providers. Adapters self-register via `init()`.

```go
// In adapter's init()
func init() {
    registry.RegisterDatabaseBuildFromEnv("firestore", BuildFromEnv)
    registry.RegisterRepositoryFactory("firestore", "client", NewClientRepo)
}
```

### 2. InstanceRegistry (Runtime)
Manages **live provider instances** with health checks and cleanup.

```go
// At runtime
reg := registry.NewRegistry()
reg.RegisterDatabaseProvider(firestoreProvider)
reg.HealthCheck(ctx)  // checks all enabled providers
reg.Close()           // cleanup on shutdown
```

## Directory Structure

```
registry/
├── registry.go      # Generic FactoryRegistry[T, C] - core generic type
├── instances.go     # InstanceRegistry[T] + Registry runtime manager
├── database.go      # Database factories + TableConfig + Repository + Operations
├── auth.go          # Auth provider factories
├── storage.go       # Storage provider factories
├── email.go         # Email provider factories
├── payment.go       # Payment provider factories
├── id.go            # ID service factories + IDProviderConfig
├── translation.go   # Translation service factories + TranslationProviderConfig
└── convenience.go   # ListAllAvailable*() helpers
```

## Registration Flow

```
1. Go program starts, init() functions run
                              │
                              ▼
2. Each adapter registers its factory:
   - RegisterDatabaseBuildFromEnv("firestore", buildFn)
   - RegisterRepositoryFactory("firestore", "client", repoFn)
   - RegisterDatabaseOperationsFactory("firestore", opsFn)
                              │
                              ▼
3. Composition layer requests provider by name:
   - BuildDatabaseProviderFromEnv("firestore")
   - CreateRepository("firestore", "client", conn, tableName)
                              │
                              ▼
4. Registry looks up and invokes registered factory
                              │
                              ▼
5. Provider instance returned, ready for use
```

## Key Components

### registry.go
Generic `FactoryRegistry[T, C]` providing:
- `RegisterFactory()` / `GetFactory()` - basic factory storage
- `RegisterConfigTransformer()` - raw config → protobuf conversion
- `RegisterBuildFromEnv()` / `BuildFromEnv()` - self-configuring providers

### database.go
Database-specific registries (largest file due to 40+ repositories):
- **Provider factories** - firestore, postgres, mock database providers
- **TableConfigBuilder** - per-provider table/collection naming
- **RepositoryFactory** - `provider:entity` keyed repository creation
- **DatabaseOperationsFactory** - CRUD operations abstraction

### instances.go
Runtime instance management:
- `InstanceRegistry[T]` - generic instance storage with health checks
- `Registry` - aggregates all provider types (database, auth, storage, email, payment)
- `HealthCheck()` - checks all enabled providers
- `Close()` - graceful shutdown of all providers

### Provider Files (auth.go, storage.go, email.go, payment.go, id.go, translation.go)
Each contains:
- Global registry instance (`var authRegistry = NewFactoryRegistry[...]`)
- Public registration functions (`RegisterAuthProviderFactory`)
- Public retrieval functions (`GetAuthProviderFactory`, `BuildAuthProviderFromEnv`)

## Provider Types (7 Total)

| Provider | Config Type | Example Implementations |
|----------|-------------|-------------------------|
| database | `*dbpb.DatabaseProviderConfig` | firestore, postgres, mock |
| auth | `*authpb.ProviderConfig` | firebase, mock |
| storage | `*storagepb.StorageProviderConfig` | gcs, s3, azure, mock |
| email | `*emailpb.EmailProviderConfig` | gmail, microsoft, mock |
| payment | `*paymentpb.PaymentProviderConfig` | maya, asiapay, mock |
| id | `*IDProviderConfig` | uuidv7, noop |
| translation | `*TranslationProviderConfig` | file, lyngua, noop, mock |

## Usage Patterns

### Registering a New Adapter
```go
// In your adapter file (e.g., adapters/secondary/database/sqlite/adapter.go)
func init() {
    registry.RegisterDatabaseBuildFromEnv("sqlite", func() (ports.DatabaseProvider, error) {
        // read env vars, create provider
        return NewSQLiteProvider(), nil
    })
}
```

### Retrieving a Provider
```go
// In composition layer
provider, err := registry.BuildDatabaseProviderFromEnv("firestore")

// Or get factory for manual initialization
factory, exists := registry.GetDatabaseProviderFactory("postgres")
if exists {
    provider := factory()
    provider.Initialize(config)
}
```

### Creating Repositories
```go
// Composition layer creates all 40 repositories dynamically
for _, entity := range []string{"client", "user", "product", ...} {
    repo, err := registry.CreateRepository("firestore", entity, conn, tableConfig[entity])
}
```

## Import Pattern

```go
import "leapfor.xyz/espyna/internal/infrastructure/registry"

// Registration (in adapters)
registry.RegisterDatabaseBuildFromEnv("mydb", buildFn)

// Retrieval (in composition)
provider, err := registry.BuildDatabaseProviderFromEnv("mydb")
```

## Key Design Decisions

1. **Self-registration via init()** - Adapters register themselves, no central switch statement
2. **Go generics** - `FactoryRegistry[T, C]` eliminates duplication across 7 provider types
3. **Composite keys** - Repository factories use `provider:entity` keys (e.g., `"firestore:client"`)
4. **Separation of concerns** - Factory registry (compile-time) vs Instance registry (runtime)
5. **Thread-safe** - All registries use `sync.RWMutex` for concurrent access
