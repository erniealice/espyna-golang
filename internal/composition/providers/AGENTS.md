# Agent Notes: Composition Layer Providers

**Objective**: To clarify the role of the composition layer and its sub-registries.

---

## 1. Core Principles of this Layer

*   **Concern**: Runtime orchestration. This layer decides **which** provider implementations are **active** based on runtime configuration.
*   **Responsibility**: To select from the available infrastructure "blueprints" (factories), construct live provider instances, configure them, and manage their lifecycle.
*   **Single Source of Truth**: The `registry.go` in this directory orchestrates all sub-registries and is the single source of truth for all *active* provider instances.

---

## 2. Current Structure

```
providers/
├── registry.go              # Main orchestrator - calls sub-registries
├── manager.go               # Provider lifecycle management
├── README.md
│
├── infrastructure/          # Core infrastructure providers
│   ├── registry.go          # Orchestrates database, auth, storage, id
│   ├── common.go            # ProviderWrapper adapter
│   ├── database.go          # CreateDatabaseProvider()
│   ├── auth.go              # CreateAuthProvider()
│   ├── storage.go           # CreateStorageProvider()
│   └── id.go                # CreateIDProvider() + IDProviderAdapter
│
├── domain/                  # Domain repository factories
│   ├── registry.go          # Orchestrates all domain repositories
│   ├── entity.go            # EntityRepositories (19 repos)
│   ├── subscription.go      # SubscriptionRepositories
│   ├── payment.go           # PaymentRepositories
│   ├── product.go           # ProductRepositories
│   ├── event.go             # EventRepositories
│   ├── workflow.go          # WorkflowRepositories
│   └── common.go            # CommonRepositories
│
└── integration/             # External service providers
    ├── registry.go          # Orchestrates email, payment
    ├── email.go             # CreateEmailProvider()
    └── payment.go           # CreatePaymentProvider()
```

---

## 3. Directory Responsibilities

| Directory | Purpose | Pattern |
|---|---|---|
| `registry.go` | **Main Orchestrator**: Initializes all sub-registries from config | Facade pattern |
| `infrastructure/` | **Core Providers**: Database, Auth, Storage, ID | Config-based factory selection |
| `domain/` | **Repository Factories**: Creates domain-specific repository collections | Lazy initialization |
| `integration/` | **External Services**: Email, Payment providers | Config-based factory selection |

---

## 4. Provider Creation Pattern

All provider creation follows the same pattern:

```go
// Example: CreateDatabaseProvider in infrastructure/database.go
func CreateDatabaseProvider(config options.DatabaseConfig) (types.Provider, error) {
    // 1. Determine provider name from config
    var providerName string
    if config.Postgres != nil {
        providerName = "postgresql"
    } else if config.Mock != nil {
        providerName = "mock"
    }

    // 2. Get factory from infrastructure registry
    factory, exists := infraregistry.GetDatabaseProviderFactory(providerName)
    if !exists {
        return nil, fmt.Errorf("no factory for: %s", providerName)
    }

    // 3. Create and initialize provider
    provider := factory()
    provider.Initialize(config)

    // 4. Wrap for composition interface compatibility
    return &ProviderWrapper{provider: provider}, nil
}
```

---

## 5. Usage Flow

```
ManagerConfig
    ↓
Registry.InitializeAll(config)
    ├─→ infrastructure.Registry.InitializeAll()
    │   ├─→ CreateDatabaseProvider()
    │   ├─→ CreateAuthProvider()
    │   ├─→ CreateStorageProvider()
    │   └─→ CreateIDProvider()
    │
    ├─→ domain.Registry = NewRegistry(dbProvider, tableConfig)
    │   └─→ InitializeAll() creates all repository collections
    │
    └─→ integration.Registry.InitializeAll()
        ├─→ CreateEmailProvider()
        └─→ CreatePaymentProvider()
```

---

## 6. Accessing Providers

```go
registry := providers.NewRegistry()
registry.InitializeAll(config, dbTableConfig)

// Infrastructure providers
dbProvider := registry.GetDatabase()
authProvider := registry.GetAuth()
idService := registry.GetIDService()

// Domain repositories
entityRepos, _ := registry.GetDomain().GetEntity()
subscriptionRepos, _ := registry.GetDomain().GetSubscription()

// Integration providers
emailProvider := registry.GetEmail()
paymentProvider := registry.GetPayment()
```

---

## 7. Adding New Providers

### Adding a new infrastructure provider (e.g., cache):

1. Add `CacheConfig` to `options/integration.go`
2. Create `infrastructure/cache.go` with `CreateCacheProvider()`
3. Add `cache types.Provider` to `infrastructure/registry.go`
4. Add getter method `GetCache()` to main `registry.go`

### Adding a new integration provider (e.g., SMS):

1. Add `SMSConfig` to `options/integration.go`
2. Create `integration/sms.go` with `CreateSMSProvider()`
3. Add `sms ports.SMSProvider` to `integration/registry.go`
4. Add getter method `GetSMS()` to main `registry.go`

### Adding a new domain repository collection:

1. Create `domain/newdomain.go` with `NewDomainRepositories` struct
2. Add field and getter to `domain/registry.go`
