# Routing Configuration Package

This package defines HTTP route configurations for the Espyna backend, organizing routes by domain and responsibility layer.

## Directory Structure

```
config/
├── config.go                    # Entry point - aggregates all configurations
├── domain/                      # Business domain route configurations
│   ├── common.go               # Common domain (Attribute)
│   ├── entity.go               # Entity domain (19 modules)
│   ├── event.go                # Event domain (3 modules)
│   ├── payment.go              # Payment domain (4 modules)
│   ├── product.go              # Product domain (9 modules)
│   ├── subscription.go         # Subscription domain (10 modules)
│   └── workflow.go             # Workflow domain (6 modules)
├── integration/                 # External service integrations
│   ├── email.go                # Gmail integration (build tag: google && gmail)
│   ├── email_stub.go           # Disabled stub (without tags)
│   ├── payment.go              # AsiaPay integration (build tag: asiapay)
│   └── payment_stub.go         # Disabled stub (without tag)
└── orchestration/               # Workflow engine routes
    └── engine.go               # Engine operations (start, continue, status)
```

## Architecture Pattern

This package follows Clean Architecture principles:

1. **Domain Layer** (`domain/`): Exposes CRUD operations for domain entities via protobuf-defined request/response types
2. **Integration Layer** (`integration/`): Handles external provider integrations with conditional compilation via Go build tags
3. **Orchestration Layer** (`orchestration/`): Coordinates complex workflows through the engine service

## Entry Point

### `config.go`

- **Function**: `GetAllDomainConfigurations(useCases, engineService)`
- **Purpose**: Aggregates all domain configurations and returns them as a slice
- **Conditional Logic**:
  - Domain routes: Always included (if use cases available)
  - Integration routes: Added only if `useCases.Integration` is non-nil AND build tags match
  - Orchestration routes: Added only if `engineService` is non-nil

## Domain Configurations

Each domain file exports a `Configure*Domain` function that:
1. Receives use cases from the application layer
2. Returns a `contracts.DomainRouteConfiguration` struct
3. Gracefully handles nil use cases (returns disabled config)

### Route Pattern
All routes follow the pattern: `POST /api/{domain}/{module}/{operation}`

**Standard operations per module:**
- `create` - Create entity
- `read` - Read single entity
- `update` - Update entity
- `delete` - Delete entity
- `list` - List entities with pagination
- `get-list-page-data` - Frontend list page optimization
- `get-item-page-data` - Frontend item page optimization

### Domain Module Counts

| Domain | Modules | Route Count |
|--------|---------|-------------|
| Common | 1 (Attribute) | 5 |
| Entity | 19 | ~133 |
| Event | 3 | 21 |
| Payment | 4 | 20 |
| Product | 9 | 59 |
| Subscription | 10 | 67 |
| Workflow | 6 | 37 |

## Integration Configurations

Integration routes use **Go build tags** for conditional compilation:

### Email Integration
- **Build tags**: `google && gmail`
- **Routes**: `/integration/email/{send,health,capabilities}`
- **Stub**: Returns disabled config when tags not present

### Payment Integration
- **Build tags**: `asiapay`
- **Routes**: `/integration/payment/{webhook,checkout,status,health,capabilities}`
- **Stub**: Returns disabled config when tag not present

## Orchestration Configuration

### `orchestration/engine.go`

Exposes workflow engine operations via adapter wrappers:

| Route | Adapter | Description |
|-------|---------|-------------|
| `/api/workflow/engine/start` | `startWorkflowAdapter` | Start workflow from template |
| `/api/workflow/engine/status` | `getWorkflowStatusAdapter` | Get workflow status |
| `/api/workflow/engine/continue` | `continueWorkflowAdapter` | Continue paused workflow |
| `/api/workflow/engine/execute-activity` | `executeActivityAdapter` | Execute specific activity |
| `/api/workflow/engine/advance` | `advanceWorkflowAdapter` | Advance to next stage |

**Adapter Pattern**: Each adapter wraps a `ports.WorkflowEngineService` method to implement the `UseCaseExecutor` interface expected by `contracts.NewGenericHandler`.

## Key Contracts Used

- `contracts.DomainRouteConfiguration` - Domain configuration with routes
- `contracts.RouteConfiguration` - Single route (Method, Path, Handler)
- `contracts.NewGenericHandler` - Creates handler from use case + request prototype

## Adding New Routes

1. **New Domain Module**: Add routes to existing domain file or create new domain file
2. **New Integration**: Create file with build tag, plus stub file without tag
3. **New Orchestration**: Add adapter wrapper + route in `engine.go`

## Dependencies

- `leapfor.xyz/espyna/internal/application/usecases/*` - Use case aggregates
- `leapfor.xyz/espyna/internal/application/ports` - Port interfaces
- `leapfor.xyz/espyna/internal/composition/contracts` - Route contracts
- `leapfor.xyz/esqyma/golang/v1/*` - Protobuf request/response types
