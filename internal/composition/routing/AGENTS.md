# routing/

Framework-agnostic HTTP routing system that composes routes from domain use cases.

## Architecture Overview

```
                            ┌──────────────────────────────────────────┐
                            │            Consumer App                   │
                            │     (apps/be-master, etc.)               │
                            └─────────────────┬────────────────────────┘
                                              │
                                              ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              ROUTING LAYER                                    │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                           Composer                                      │  │
│  │                    (orchestrates everything)                            │  │
│  └────────────────────────────┬───────────────────────────────────────────┘  │
│                               │                                              │
│         ┌─────────────────────┼─────────────────────┐                       │
│         ▼                     ▼                     ▼                       │
│  ┌─────────────┐     ┌───────────────┐     ┌──────────────────┐            │
│  │RouteManager │     │ config/domain │     │  customization/  │            │
│  │(registration│     │ (route defs)  │     │(consumer tweaks) │            │
│  │& retrieval) │     └───────────────┘     └──────────────────┘            │
│  └─────────────┘                                                            │
└──────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                       HTTP Framework Adapters                                 │
│            (infrastructure/adapters/primary/http/gin, fiber, etc.)           │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
routing/
├── AGENTS.md              # This file
├── contracts.go           # Type aliases + builders + routing-specific types
├── manager.go             # RouteManager - route registration & retrieval
├── composer.go            # Composer - orchestrates route initialization
├── config/                # Route configurations by domain
│   ├── domain.go          # GetAllDomainConfigurations() aggregator
│   ├── domain/            # Domain-specific route configs
│   │   ├── common.go      # Attribute routes
│   │   ├── entity.go      # User, Client, Manager, Delegate routes
│   │   ├── event.go       # Event scheduling routes
│   │   ├── payment.go     # Payment processing routes
│   │   ├── product.go     # Product catalog routes
│   │   ├── subscription.go# Plan, Invoice, Balance routes
│   │   └── workflow.go    # Workflow engine routes
│   └── integration/       # Integration route configs
│       ├── email.go       # Email provider routes
│       └── payment.go     # Payment provider webhooks
├── customization/         # Consumer route customization
│   ├── types.go           # RouteCustomizer struct
│   └── customizer.go      # Path prefix customization
└── handlers/              # Handler adapters
    ├── infrastructure.go  # Infrastructure handlers
    └── integration.go     # Integration handlers
```

## Type System

### Contracts (Source of Truth)
All core routing types are defined in `contracts/` and re-exported here as aliases:

```go
// From contracts/ (single source of truth)
type Handler = contracts.RouteHandler      // Execute(ctx, proto.Message) -> proto.Message
type Route = contracts.Route               // Method, Path, Handler, Middleware, Metadata
type RouteGroup = contracts.RouteGroup     // Prefix, Routes, SubGroups
type Config = contracts.Config             // Routing configuration with methods
```

### Routing-Specific Types (This Package)
Types that contain mutable state or application-layer dependencies:

```go
// Builders (have methods defined in interfaces.go)
type RouteBuilder struct { route *Route }
type GroupBuilder struct { group *RouteGroup }

// Managers (have sync.RWMutex, manage runtime state)
type RouteManager struct { ... }
type MigrationManager struct { ... }

// Composer (imports application layer)
type Composer struct {
    useCases *usecases.Aggregate  // This is why it can't be in contracts
    ...
}
```

## Key Components

### 1. RouteManager (`manager.go`)

Manages route registration, organization, and retrieval:

```go
// Create a route manager
rm := NewRouteManager(config)

// Register routes
rm.RegisterRoute("entity_user_create", route)
rm.RegisterGroup(userGroup)

// Retrieve routes
route := rm.GetRoute("entity_user_create")
routes := rm.GetRoutesByDomain("entity")
allRoutes := rm.GetAllRoutes()
```

**Key Methods:**
- `RegisterRoute(name, route)` - Register a single route
- `RegisterGroup(group)` - Register a route group
- `GetAllRoutes()` - Get all registered routes
- `GetRoutesByDomain(domain)` - Filter routes by domain
- `DefaultConfig()` - Get default routing configuration

### 2. Composer (`composer.go`)

Orchestrates the entire routing system initialization:

```go
composer, err := NewComposer(&ComposerConfig{
    Config:    routingConfig,
    Container: container,  // DI container with use cases
})

// Get composed routes for HTTP adapter
routes := composer.GetAllRoutes()
```

**Initialization Flow:**
1. Creates RouteManager with configuration
2. Creates MigrationManager for legacy support
3. Extracts use cases from container
4. Calls `initializeRoutes()` to wire everything

### 3. Route Configuration (`config/`)

Each domain has a configuration function that wires use cases to routes:

```go
// config/domain/entity.go
func ConfigureEntityDomain(useCases *entity.EntityUseCases) contracts.DomainRouteConfiguration {
    return contracts.DomainRouteConfiguration{
        Domain:  "entity",
        Prefix:  "/entity",
        Enabled: useCases != nil,
        Routes: []contracts.RouteConfiguration{
            {Method: "GET", Path: "/users", Handler: useCases.User.List},
            {Method: "POST", Path: "/users", Handler: useCases.User.Create},
            // ... more routes
        },
    }
}
```

**Aggregator Function:**
```go
// config/domain.go
func GetAllDomainConfigurations(useCases *usecases.Aggregate) []contracts.DomainRouteConfiguration {
    return []contracts.DomainRouteConfiguration{
        domain.ConfigureCommonDomain(useCases.Common),
        domain.ConfigureEntityDomain(useCases.Entity),
        domain.ConfigureEventDomain(useCases.Event),
        domain.ConfigurePaymentDomain(useCases.Payment),
        domain.ConfigureProductDomain(useCases.Product),
        domain.ConfigureSubscriptionDomain(useCases.Subscription),
        domain.ConfigureWorkflowDomain(useCases.Workflow),
    }
}
```

### 4. Route Customization (`customization/`)

Allows consumer apps to customize route paths:

```go
customizer := customization.NewRouteCustomizer().
    WithGlobalPrefix("/api/v2").           // All routes: /api/v2/...
    WithDomainPrefix("entity", "/users").  // Entity routes: /api/v2/users/...
    WithRoutePath("entity_user_create", "/signup")  // Override specific route

customizedRoutes := customizer.ApplyCustomizations(routes)
```

### 5. Builder Pattern (`interfaces.go`)

Fluent interface for building routes programmatically:

```go
// Build a single route
route := NewRouteBuilder("POST", "/users").
    Handler(createUserHandler).
    Middleware(authMiddleware, loggingMiddleware).
    Metadata(RouteMetadata{
        Domain:    "entity",
        Resource:  "user",
        Operation: "create",
    }).
    Build()

// Build a route group
group := NewGroupBuilder("/api").
    Route("GET", "/health", healthHandler).
    Route("GET", "/version", versionHandler).
    SubGroup("/users").
        Route("GET", "/", listUsersHandler).
        Route("POST", "/", createUserHandler).
    Build()
```

## How Routes Flow to HTTP Adapters

```
1. Container initializes use cases
              │
              ▼
2. Composer receives container
              │
              ▼
3. config.GetAllDomainConfigurations(useCases)
   returns []DomainRouteConfiguration
              │
              ▼
4. Composer converts to []*Route via initializeRoutes()
              │
              ▼
5. RouteManager stores routes
              │
              ▼
6. HTTP adapter (Gin/Fiber/etc.) calls GetAllRoutes()
              │
              ▼
7. Adapter converts routing.Route -> framework-specific handlers
```

## Configuration

The `Config` type controls routing behavior:

```go
config := &Config{
    BasePath:          "/api",
    Timeout:           30 * time.Second,
    EnableAuth:        true,
    EnableMetrics:     true,
    EnableHealthCheck: true,
    CORS: CORSConfig{
        Enabled:        true,
        AllowedOrigins: []string{"*"},
    },
    Domains: map[string]DomainConfig{
        "entity": {Enabled: true, Prefix: "/entity"},
        "event":  {Enabled: true, Prefix: "/event"},
    },
}
```

**Config Methods (defined in contracts/config.go):**
- `Validate()` - Validates and sets defaults
- `GetEnabledDomains()` - Returns list of enabled domains
- `IsEndpointEnabled(domain, endpoint)` - Checks if endpoint is enabled

## Import Patterns

```go
// For route types (aliases to contracts)
import "leapfor.xyz/espyna/internal/composition/routing"
route := &routing.Route{...}

// For route configuration functions
import "leapfor.xyz/espyna/internal/composition/routing/config"
configs := config.GetAllDomainConfigurations(useCases)

// For customization
import "leapfor.xyz/espyna/internal/composition/routing/customization"
customizer := customization.NewRouteCustomizer()
```

## Relationship to contracts/

This package **uses** contracts as the source of truth:

| This Package | contracts/ |
|--------------|------------|
| `routing.Handler` (alias) | `contracts.RouteHandler` (definition) |
| `routing.Route` (alias) | `contracts.Route` (definition) |
| `routing.Config` (alias) | `contracts.Config` (definition + methods) |
| `routing.RouteManager` (local) | - |
| `routing.Composer` (local) | - |

The separation ensures:
- contracts/ has no dependencies on routing implementation
- contracts/ can be imported by infrastructure adapters
- routing/ contains all stateful/mutable components
