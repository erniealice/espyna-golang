# Routing - HTTP Routing System

The routing layer provides a framework-agnostic HTTP routing system that manages all application routes and handlers in a clean, organized manner.

## Overview

This directory contains the routing system that separates route management from HTTP framework-specific implementations. It follows hexagonal architecture principles by keeping routing logic framework-agnostic.

**Current Implementation**:
- `RouteManager` provides framework-agnostic route registration
- Includes `Close()` method for lifecycle management
- Integrates with `infrastructure/adapters/primary/http/registry` for backward compatibility
- Route configuration via `routing.Config` from options package

## Components

### `route_manager.go`
Framework-agnostic route manager that handles route registration, middleware chain, and handler dispatch. This component:
- Manages route registration and discovery
- Handles middleware chain composition
- Provides framework-agnostic routing interfaces
- Supports route grouping and organization
- Enables runtime route configuration

### `route_config.go`
Route configuration types and structures that define routing behavior, including:
- Route definitions and metadata
- Middleware configurations
- Route group configurations
- Security and authentication settings
- Versioning and API configuration

### `handlers/`
Organized handler components grouped by responsibility:

#### `handlers/domain.go`
Domain-specific business logic handlers:
- **Entity handlers**: Users, clients, groups, roles, permissions
- **Event handlers**: Event management and processing
- **Framework handlers**: Frameworks, objectives, tasks
- **Payment handlers**: Payments, invoices, subscriptions
- **Product handlers**: Products, collections, pricing
- **Record handlers**: Record management
- **Subscription handlers**: Plans, subscriptions, billing

#### `handlers/infrastructure.go`
Infrastructure and system-level handlers:
- **Health checks**: Application and dependency health
- **Metrics**: Application performance monitoring
- **System**: System information and diagnostics
- **Configuration**: Runtime configuration management
- **Logging**: Log management and inspection

#### `handlers/integration.go`
External integration handlers:
- **Webhooks**: Incoming webhook processing
- **External APIs**: Third-party service integrations
- **File upload**: File handling and processing
- **Notifications**: Push and email notifications
- **Callbacks**: Asynchronous callback handling

## Architecture Principles

- **Framework Independence**: Routing logic is completely framework-agnostic
- **Clean Separation**: Handlers organized by responsibility and domain
- **Middleware Support**: Comprehensive middleware chain management
- **Configuration-Driven**: Routes configured through functional options
- **Gradual Migration**: Supports coexistence with legacy routing

## Handler Organization

### Domain Handlers
```go
// Domain handlers follow business domain structure
type DomainHandlers struct {
    EntityHandlers    *EntityHandlers
    EventHandlers     *EventHandlers
    PaymentHandlers   *PaymentHandlers
    ProductHandlers   *ProductHandlers
    SubscriptionHandlers *SubscriptionHandlers
}
```

### Infrastructure Handlers
```go
// Infrastructure handlers handle system concerns
type InfrastructureHandlers struct {
    HealthHandler  *HealthHandler
    MetricsHandler *MetricsHandler
    SystemHandler  *SystemHandler
}
```

### Integration Handlers
```go
// Integration handlers handle external systems
type IntegrationHandlers struct {
    WebhookHandler *WebhookHandler
    APIHandler     *ExternalAPIHandler
    FileHandler    *FileHandler
}
```

## Usage

```go
// Create route manager
routeManager := NewRouteManager(&RouteConfig{
    BasePath:     "/api/v1",
    EnableAuth:   true,
    EnableMetrics: true,
})

// Register handlers
routeManager.RegisterDomainHandlers(domainHandlers)
routeManager.RegisterInfrastructureHandlers(infrastructureHandlers)
routeManager.RegisterIntegrationHandlers(integrationHandlers)

// Setup routes for specific framework
ginAdapter := NewGinAdapter(engine)
ginAdapter.SetupRoutes(routeManager)
```

## Framework Integration

The routing system supports multiple HTTP frameworks through adapter pattern:

### Gin Framework
```go
ginAdapter := &GinRouteWrapper{
    engine: gin.New(),
    routeManager: routeManager,
}
```

### Fiber Framework
```go
fiberAdapter := &FiberRouteWrapper{
    app: fiber.New(),
    routeManager: routeManager,
}
```

### Vanilla HTTP
```go
vanillaAdapter := &VanillaRouteWrapper{
    mux: http.NewServeMux(),
    routeManager: routeManager,
}
```

## Integration

This layer integrates with:
- **Core Layer**: Route manager is injected into the container
- **Options Layer**: Route configuration comes from functional options
- **Infrastructure Layer**: HTTP adapters implement the routing interfaces

## Benefits

- **Framework Agnostic**: Same routing logic works with any HTTP framework
- **Organized Handlers**: Clear separation of concerns
- **Middleware Support**: Comprehensive middleware management
- **Gradual Migration**: Can coexist with existing routing systems
- **Testability**: Easy to test routing logic without HTTP framework

## Migration Strategy

This routing system is designed to work alongside existing routes during gradual migration:

1. **Coexistence**: New routing system works alongside legacy routes
2. **Gradual Migration**: Routes migrated one by one
3. **Feature Flags**: Control which system handles specific routes
4. **Safe Rollback**: Easy to revert if issues arise

## Configuration

```go
type RoutingConfig struct {
    UseNewRouting   bool            // Global switch for new routing
    BasePath        string          // Base path for all routes
    EnableAuth      bool            // Enable authentication middleware
    EnableMetrics   bool            // Enable metrics middleware
    HandlerGroups   []string        // Active handler groups
    MigratedRoutes  []string        // Routes using new system
    LegacyRoutes    []string        // Routes using legacy system
}
```