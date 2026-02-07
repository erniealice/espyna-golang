# composition/

The **Composition Root** - orchestrates dependency injection and wires together all application components following hexagonal architecture principles.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CONSUMER APP                                    │
│                         (apps/be-master, etc.)                              │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           consumer/consumer.go                               │
│                    NewContainer() / NewContainerFromEnv()                    │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              COMPOSITION LAYER                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         core/container.go                            │   │
│  │                    Main DI Container & Orchestrator                  │   │
│  └───────────────────────────────────┬─────────────────────────────────┘   │
│                                      │                                      │
│         ┌────────────────────────────┼────────────────────────────┐        │
│         ▼                            ▼                            ▼        │
│  ┌─────────────┐           ┌─────────────────┐           ┌─────────────┐  │
│  │   options/  │           │   providers/    │           │   routing/  │  │
│  │ Config Types│──────────▶│  Provider Mgmt  │           │ HTTP Routes │  │
│  └─────────────┘           └────────┬────────┘           └──────┬──────┘  │
│                                     │                           │         │
│                                     └───────────┬───────────────┘         │
│                                                 ▼                         │
│                                    ┌────────────────────┐                 │
│                                    │ application/usecases│                 │
│                                    │  (injected at init) │                 │
│                                    └────────────────────┘                 │
│                                      │                                      │
│                    ┌─────────────────┼─────────────────┐                   │
│                    ▼                 ▼                 ▼                   │
│           ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│           │infrastructure│  │    domain    │  │ integration  │            │
│           │  (db, auth)  │  │ (repositories)│  │(email, pay) │            │
│           └──────────────┘  └──────────────┘  └──────────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          APPLICATION LAYER                                   │
│                    internal/application/usecases/                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INFRASTRUCTURE LAYER                                 │
│                    internal/infrastructure/providers/                        │
│                  (PostgreSQL, Firestore, Firebase, etc.)                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Dependency Flow (All Unidirectional)

```
                    ┌─────────────┐
                    │  contracts/ │ ◀──── shared interfaces, no deps
                    └──────┬──────┘
                           │
       ┌───────────────────┼───────────────────┐
       ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   options/  │    │  providers/ │    │   routing/  │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                          ▼
                 ┌─────────────────┐
                 │core/container.go│ ◀──── orchestrates all, imports all
                 └─────────────────┘
```

**Key insight:** `providers/` and `routing/` do **NOT** import each other.
- `options/` → consumed by `core/container.go` (config flows in)
- `providers/` → creates use cases, injected into routing handlers
- `routing/` → calls use cases at runtime (not compile-time dependency)
- `core/container.go` → imports and orchestrates everything

This maintains clean hexagonal architecture - the composition root wires
dependencies without creating circular imports between siblings.

## Directory Structure

```
composition/
├── config/                 # Application-level config (DatabaseTableConfig)
├── contracts/              # DDD interfaces & shared types
├── core/                   # Main container & orchestration
│   └── initializers/       # Domain-specific use case initialization
├── options/                # Functional options pattern (configuration)
│   ├── infrastructure/     # Core system options (db, auth, storage)
│   └── integrations/       # External service options (email, payment)
├── providers/              # Provider management & creation
│   ├── domain/             # Domain repository providers (7 domains)
│   ├── infrastructure/     # Infrastructure provider factories
│   └── integration/        # Integration provider factories
└── routing/                # HTTP routing system
    ├── config/             # Route definitions by domain
    ├── customization/      # Consumer route customization
    └── handlers/           # Route handlers
```

## Initialization Flow

```
1. Consumer calls NewContainerFromEnv() or NewContainer(opts...)
                              │
                              ▼
2. core/container.go creates Container with Config
   - Reads CONFIG_* environment variables
   - Sets up infrastructure configs (database, auth, storage, id)
                              │
                              ▼
3. providers/manager.go creates Provider Manager
   - Creates infrastructure providers via providers/infrastructure/
   - Creates domain repositories via providers/domain/
   - Creates integration providers via providers/integration/
                              │
                              ▼
4. core/initializers/*.go initialize domain-specific use cases
   - entity.go → User, Client, Manager, Delegate use cases
   - event.go → Event scheduling use cases
   - payment.go → Payment processing use cases
   - product.go → Product catalog use cases
   - subscription.go → Plan, Invoice, Balance use cases
   - workflow.go → Workflow engine use cases
                              │
                              ▼
5. routing/manager.go sets up HTTP routes
   - Loads route configs from routing/config/domain/*.go
   - Composes routes via routing/composer.go
   - Applies consumer customizations via routing/customization/
                              │
                              ▼
6. Container ready - exposes UseCases, Providers, Routes
```

## Key Components

### contracts/
DDD-style interfaces defining behavioral boundaries:
- `Service`, `UseCase`, `Repository`, `Provider` interfaces
- `Domain` constants (entity, event, payment, product, subscription, workflow)
- Route types (`RouteHandler`, `Route`, `RouteGroup`)

### options/
Functional options pattern for composable configuration:
- **infrastructure/** - `WithDatabaseFromEnv()`, `WithAuthFromEnv()`, etc.
- **integrations/** - `WithEmailFromEnv()`, `WithPaymentFromEnv()`, etc.
- **config.go** - `ManagerConfig` aggregating all provider configs

### providers/
Three-tier provider management:
1. **infrastructure/** - Creates database, auth, storage, ID providers
2. **domain/** - Creates 40+ repositories across 7 business domains
3. **integration/** - Creates email and payment providers

### routing/
HTTP route composition:
- **config/domain/** - Route definitions per domain (entity, event, etc.)
- **config/integration/** - Integration routes (email, payment webhooks)
- **customization/** - Consumer can add/modify/disable routes
- **handlers/** - Generic handler implementations

## Domain Mapping (7 Domains)

| Domain | Entities | Primary Use Cases |
|--------|----------|-------------------|
| entity | User, Client, Manager, Delegate, Group, Location, Role | User management |
| event | Event, EventClient, EventProduct | Scheduling |
| payment | Payment, PaymentMethod, PaymentProfile | Transactions |
| product | Product, Collection, Resource, PriceProduct | Catalog |
| subscription | Plan, Subscription, Invoice, Balance | Billing |
| workflow | Workflow, Stage, Activity, Templates | Process automation |
| common | Attribute (cross-domain) | Shared |

## Import Patterns

```go
// Container and options
import "leapfor.xyz/espyna/consumer"
import infraopts "leapfor.xyz/espyna/internal/composition/options/infrastructure"

// Contracts
import "leapfor.xyz/espyna/internal/composition/contracts"

// Providers
import "leapfor.xyz/espyna/internal/composition/providers"
```

## Environment Variables

Provider selection via `CONFIG_*` variables:
- `CONFIG_DATABASE_PROVIDER`: mock_db, postgres, firestore
- `CONFIG_AUTH_PROVIDER`: mock_auth, firebase_auth, jwt_auth
- `CONFIG_STORAGE_PROVIDER`: mock_storage, local_storage, gcs, s3
- `CONFIG_ID_PROVIDER`: noop, google_uuidv7
- `CONFIG_EMAIL_PROVIDER`: mock, gmail, microsoft
- `CONFIG_PAYMENT_PROVIDER`: mock, stripe, asiapay
