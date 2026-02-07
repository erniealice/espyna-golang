# primary/

**Inbound Adapters** (Driving Adapters) - handle how external systems talk to the application via HTTP requests, implementing routes and middleware across multiple web frameworks.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            EXTERNAL CLIENT                                   │
│                    (Browser, Mobile App, API Consumer)                       │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │ HTTP Request
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PRIMARY ADAPTERS                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Framework Server                                │   │
│  │          (fiber/server.go | gin/server.go | vanilla/server.go)      │   │
│  └───────────────────────────────────┬─────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     MIDDLEWARE CHAIN                                 │   │
│  │  ┌─────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌─────────┐  ┌───────────┐ │   │
│  │  │CORS │→ │ GZIP │→ │ CSRF │→ │ Auth │→ │Business │→ │  Authz    │ │   │
│  │  │     │  │      │  │      │  │      │  │  Type   │  │           │ │   │
│  │  └─────┘  └──────┘  └──────┘  └──────┘  └─────────┘  └───────────┘ │   │
│  └───────────────────────────────────┬─────────────────────────────────┘   │
│                                      │                                      │
│                                      ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Framework Routes                                 │   │
│  │          (fiber/routes.go | gin/routes.go | vanilla/routes.go)      │   │
│  └───────────────────────────────────┬─────────────────────────────────┘   │
│                                      │                                      │
│                       ┌──────────────┴──────────────┐                      │
│                       ▼                              ▼                      │
│              ┌──────────────┐               ┌──────────────┐               │
│              │shared/       │               │shared/       │               │
│              │request_data  │               │route_helpers │               │
│              └──────────────┘               └──────────────┘               │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        COMPOSITION LAYER                                     │
│                composition/routing/handlers/ → Use Cases                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
primary/
└── http/
    ├── fiber/                  # Fiber v2 framework
    │   ├── server.go           # Server setup & lifecycle
    │   ├── routes.go           # Route registration
    │   └── middleware/
    │       ├── authentication.go  # JWT token extraction
    │       ├── authorization.go   # Permission checking
    │       ├── business_type.go   # Tenant context injection
    │       ├── cors.go            # Cross-origin resource sharing
    │       ├── csrf.go            # CSRF protection
    │       └── gzip.go            # Response compression
    │
    ├── fiberv3/                # Fiber v3 framework (same structure)
    ├── gin/                    # Gin framework (same structure)
    ├── vanilla/                # Standard library net/http (same structure)
    │
    └── shared/                 # Framework-agnostic utilities
        ├── request_data.go     # Extract request metadata
        └── route_helpers.go    # Common route utilities
```

## Multi-Framework Support

| Framework | Package | Use Case |
|-----------|---------|----------|
| Fiber v2 | `fiber` | High performance, Express-like API |
| Fiber v3 | `fiberv3` | Next version (API changes) |
| Gin | `gin` | Popular, well-documented |
| Vanilla | `vanilla` | No dependencies, debugging |

### Framework Selection

The consumer app chooses the framework at compile time:

```go
// In consumer/server_fiber.go
import "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/fiber"

server := fiber.NewServer(container)
server.Start(":8080")
```

## Middleware Chain

Requests flow through middleware in order:

```
Request → CORS → GZIP → CSRF → Authentication → BusinessType → Authorization → Handler
```

| Middleware | Purpose | Skipped Routes |
|------------|---------|----------------|
| CORS | Allow cross-origin requests | None |
| GZIP | Compress responses | None |
| CSRF | Prevent cross-site forgery | GET, OPTIONS |
| Authentication | Extract user from JWT | Public routes |
| BusinessType | Set workspace/tenant context | Non-tenant routes |
| Authorization | Check permissions | Public routes |

## Key Files

### server.go (per framework)
```go
type Server struct {
    app       *fiber.App
    container *composition.Container
}

func NewServer(container *composition.Container) *Server
func (s *Server) Start(addr string) error
func (s *Server) Shutdown(ctx context.Context) error
```

### routes.go (per framework)
```go
func (s *Server) RegisterRoutes() {
    // Delegates to composition/routing for route definitions
    // Applies framework-specific middleware
}
```

### middleware/*.go (per framework)
```go
// Authentication extracts user from Authorization header
func Authentication(authProvider ports.AuthProvider) fiber.Handler

// Authorization checks if user has required permissions
func Authorization(authProvider ports.AuthProvider) fiber.Handler

// BusinessType extracts workspace context from X-Business-Type header
func BusinessType() fiber.Handler
```

## Request Flow Example

```
1. Client sends: POST /api/clients
   Headers: Authorization: Bearer <jwt>, X-Business-Type: <workspace-id>
                              │
                              ▼
2. CORS middleware: Adds Access-Control-* headers
                              │
                              ▼
3. Authentication middleware:
   - Extracts JWT from Authorization header
   - Validates token via AuthProvider
   - Sets user in context: ctx.Locals("user", user)
                              │
                              ▼
4. BusinessType middleware:
   - Extracts workspace ID from X-Business-Type header
   - Sets workspace in context: ctx.Locals("workspace_id", id)
                              │
                              ▼
5. Authorization middleware:
   - Gets user from context
   - Checks permission for "clients:create"
   - Returns 403 if denied
                              │
                              ▼
6. Handler (in composition/routing/handlers/):
   - Parses request body
   - Calls clientUseCase.Create(ctx, req)
   - Returns JSON response
```

## Import Patterns

```go
// Server setup (in consumer app)
import "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/fiber"

// Shared utilities
import "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/shared"
```

## Related Packages

| Package | Purpose |
|---------|---------|
| `composition/routing/` | Route definitions & handler logic |
| `composition/routing/handlers/` | Use case invocation |
| `adapters/secondary/auth/` | Auth provider implementations |
| `application/ports/` | AuthProvider interface |

## Key Design Decisions

1. **Multi-framework support** - Same middleware logic across 4 frameworks for flexibility
2. **Middleware ordering** - Security-first: CORS → Auth → Authz before business logic
3. **Context propagation** - User and workspace stored in request context via `Locals()`
4. **Shared utilities** - `http/shared/` contains framework-agnostic helpers
5. **Composition delegation** - Routes delegate to composition layer for handler logic
