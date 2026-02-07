# contracts/

DDD-style **interface definitions** that define behavioral boundaries.

## Purpose

This package contains shared types and interfaces that define *what things can do* - the contracts between different parts of the system. It serves as the **single source of truth** for types used across multiple packages.

## Contents

### Core Interfaces (`core.go`)
- `Service`, `UseCase`, `Repository`, `Handler`, `Provider`
- `Domain` constants, `ProviderType` constants

### Routing Types (`routing.go`)
- `RouteHandler` - framework-agnostic handler interface
- `Route`, `RouteMetadata`, `RouteGroup`, `GroupMetadata`
- `Request`, `Response` - HTTP request/response types
- `DomainRouteConfiguration`, `RouteConfiguration`

### Configuration Types (`config.go`)
- `Config` with methods: `Validate()`, `GetEnabledDomains()`, `IsEndpointEnabled()`
- `CORSConfig`, `RateLimitConfig`, `DomainConfig`, `EndpointConfig`
- `MigrationConfig`, `TrafficSplitConfig`, `SplitRule`

### Handler Types (`handlers.go`)
- `UseCaseHandler` and handler-related interfaces

### Infrastructure Types (`infrastructure.go`)
- `Logger`, `Cache`, `EventBus` abstractions

## Import Patterns

```go
// Direct import for types
import "leapfor.xyz/espyna/internal/composition/contracts"

// The routing package re-exports these as aliases for convenience
import "leapfor.xyz/espyna/internal/composition/routing"
// routing.Route == contracts.Route
// routing.Handler == contracts.RouteHandler
```

## Not to be confused with

**`options/`** - Contains the Functional Options pattern for *configuring* the container. That package defines `WithXxx()` functions and configuration structs, not behavioral interfaces.

**`routing/`** - Contains routing *implementation* (managers, composers, builders) that use these contracts. The routing package re-exports contracts types as aliases for backward compatibility.
