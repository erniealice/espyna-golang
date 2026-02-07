# core/

The **Container Core** - the main dependency injection container and orchestrator that wires together all application components following hexagonal architecture principles.

## File Structure

```
core/
├── AGENTS.md          # This documentation
├── container.go       # Main DI Container & Orchestrator
├── config.go          # Environment variable documentation
└── usecases.go        # Use case initialization across 7 domains
```

## Initialization Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ENTRY POINT (consumer/consumer.go)                   │
│                    NewContainerFromEnv() or NewContainer()                  │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. CONTAINER CREATION (container.go:NewContainerFromEnv)                    │
│    ┌─────────────────────────────────────────────────────────────────┐      │
│    │  • Read CONFIG_* environment variables                          │      │
│    │  • Configure DatabaseProvider (mock_db | postgres | firestore)  │      │
│    │  • Configure AuthProvider (mock_auth | firebase_auth)           │      │
│    │  • Configure StorageProvider (mock_storage | local)             │      │
│    │  • Configure IDProvider (noop | google_uuidv7)                  │      │
│    │  • Create DatabaseTableConfig from environment                  │      │
│    └─────────────────────────────────────────────────────────────────┘      │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 2. CONTAINER INITIALIZATION (container.go:Initialize)                       │
│    ┌─────────────────────────────────────────────────────────────────┐      │
│    │  a) Create Provider Manager                                     │      │
│    │     providers.NewManagerWithID(db, auth, storage, id, tableConfig)     │
│    │                                                                 │      │
│    │  b) Initialize Integration Providers (optional)                 │      │
│    │     • Email provider (Gmail, SendGrid, etc.)                    │      │
│    │     • Payment provider (Stripe, AsiaPay, etc.)                  │      │
│    │                                                                 │      │
│    │  c) Initialize Use Cases (usecases.go)                          │      │
│    │     → See "Use Case Initialization Flow" below                  │      │
│    │                                                                 │      │
│    │  d) Create Routing Composer                                     │      │
│    │     routing.NewComposer() with container reference              │      │
│    └─────────────────────────────────────────────────────────────────┘      │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ 3. CONTAINER READY                                                          │
│    Exposes: UseCases, Providers, Services, Routes                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Use Case Initialization Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                UseCaseInitializer.InitializeAll() (usecases.go)             │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│ 1. Common     │         │ 2. Entity     │         │ 3. Event      │
│ (Attribute)   │         │ (16 entities) │         │ (4 entities)  │
└───────────────┘         └───────────────┘         └───────────────┘
        │                           │                           │
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│ 4. Payment    │         │ 5. Product    │         │ 6. Subscription│
│ (4 entities)  │         │ (8 entities)  │         │ (9 entities)  │
└───────────────┘         └───────────────┘         └───────────────┘
        │                           │                           │
        └───────────────────────────┼───────────────────────────┘
                                    │
                                    ▼
                         ┌───────────────────┐
                         │ 7. Workflow       │
                         │ (6 entities)      │
                         │ + Engine (late)   │
                         └───────────────────┘
                                    │
                                    ▼
                         ┌───────────────────┐
                         │ Create Aggregate  │
                         │ with all domains  │
                         └───────────────────┘
                                    │
                                    ▼
                         ┌───────────────────┐
                         │ LATE BINDING:     │
                         │ Wire Workflow     │
                         │ Engine with       │
                         │ UseCaseRegistry   │
                         └───────────────────┘
```

## Domain Initialization Pattern

Each domain follows this pattern:

```go
// 1. Get repositories from domain provider
repos, err := domain.New<Domain>Repositories(dbProvider, tableConfig)

// 2. Get shared services (auth, tx, i18n, id)
authSvc, txSvc, i18nSvc, idSvc := uci.getServices(container)

// 3. Wire via composition initializer
useCases, err := initializers.Initialize<Domain>(repos, authSvc, txSvc, i18nSvc, idSvc)
```

**Graceful Degradation**: If a domain fails to initialize, an empty struct is used for that domain only. Other domains continue to function normally.

## Key Components

### Container (container.go)

The main DI container with:

| Field | Type | Purpose |
|-------|------|---------|
| `config` | `*Config` | All provider configurations |
| `providers` | `*providers.Manager` | Database, auth, storage, ID providers |
| `routing` | `RouteManager` | HTTP route management |
| `useCases` | `*usecases.Aggregate` | All domain use cases |
| `services` | `Services` | Core infrastructure services |

### Services struct

```go
type Services struct {
    Auth        contracts.Service     // Authentication/Authorization
    Storage     contracts.Service     // File storage
    Metrics     contracts.Service     // Monitoring
    Logger      contracts.Service     // Logging
    Cache       contracts.Service     // Caching
    Translation contracts.Service     // i18n
    Transaction contracts.Service     // DB transactions
    IDGen       contracts.Service     // UUID generation
    Email       ports.EmailProvider   // Email sending
    Payment     ports.PaymentProvider // Payment processing
}
```

### Config struct

```go
type Config struct {
    // Application metadata
    Name, Version, Environment string

    // Infrastructure providers
    DatabaseProvider, AuthProvider, StorageProvider, IDProvider

    // Integration providers
    EmailProvider, PaymentProvider

    // Table naming
    DatabaseTableConfig

    // HTTP routing
    RoutingConfig
}
```

## Environment Variables

See `config.go` for complete documentation. Quick reference:

| Variable | Values | Default |
|----------|--------|---------|
| `CONFIG_DATABASE_PROVIDER` | mock_db, postgres, firestore | mock_db |
| `CONFIG_AUTH_PROVIDER` | mock_auth, firebase_auth | mock_auth |
| `CONFIG_STORAGE_PROVIDER` | mock_storage, local | mock_storage |
| `CONFIG_ID_PROVIDER` | noop, google_uuidv7 | noop |

## Container Lifecycle

```
┌──────────┐    ┌─────────────┐    ┌──────────┐    ┌────────┐
│ NewXXX() │───▶│ Initialize()│───▶│ Running  │───▶│ Close()│
└──────────┘    └─────────────┘    └──────────┘    └────────┘
                     │                   │              │
                     ▼                   ▼              ▼
              Sets initialized    GetUseCases()   Shuts down
              = true              GetServices()   providers
                                  GetRoutes()     and routes
```

## Thread Safety

All Container methods use `sync.RWMutex`:
- Read methods (`Get*`) use `RLock()`
- Write methods (`Set*`, `Initialize`, `Close`) use `Lock()`

## Late Binding: Workflow Engine

The Workflow Engine requires the complete `usecases.Aggregate` to execute activities across domains. It's initialized AFTER all domain use cases:

```go
// After all domains initialized...
aggregate := &usecases.Aggregate{...all domains...}
container.useCases = aggregate

// THEN wire the engine
useCaseRegistry := registry.NewWorkflowUseCaseRegistry(aggregate)
engineUC := initializers.InitializeWorkflowEngine(repos, services, useCaseRegistry)
container.useCases.Workflow.SetEngine(engineUC)
```

## Import Dependencies

```
core/container.go imports:
├── options/infrastructure/     (provider configs)
├── options/integrations/       (email, payment configs)
├── providers/                  (manager, domain, integration)
├── contracts/                  (interfaces)
├── routing/                    (HTTP routing)
├── config/                     (DatabaseTableConfig)
└── application/usecases/       (aggregate, domain use cases)
```

## Entry Points

| Function | Use Case |
|----------|----------|
| `NewContainer()` | Create empty container for manual configuration |
| `NewContainerFromEnv()` | **Recommended** - Auto-configure from environment |
| `NewContainerWithOptions()` | Functional options pattern |
