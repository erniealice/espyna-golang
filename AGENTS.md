# Espyna Package - Claude AI Assistant Instructions

## Package Context
**Espyna** is a Go-based hexagonal architecture package for comprehensive business management APIs. Built with gRPC/protobuf, following clean architecture principles with use cases, providers, adapters, and a sophisticated registry system.

## Quick Reference - Schema Documentation

For protobuf schema context, see the consolidated knowledge files in `packages/esqyma/schema/v1/`:

| File | Description |
|------|-------------|
| `domain.proto` | All 58+ domain models (entity, product, subscription, payment, event, workflow) |
| `infrastructure.proto` | Auth, database, storage provider schemas |
| `integration.proto` | Email, payment gateway, webhook integrations |
| `orchestration.proto` | Workflow engine service definitions |

## Production Architecture Principles

### Less is More in Go Backend Design
Following **production-ready, scalable, maintainable** Go architecture:

**Minimal, Focused Components**
- Pure hexagonal architecture - No framework coupling in business logic
- Single-purpose use cases - Each use case handles one business operation
- Explicit dependencies - Repository injection makes dependencies clear
- Standardized patterns - Same CRUD pattern across all 40+ entities

**Strategic Simplicity**
- Registry pattern over factories - Centralized component management
- Foreign key validation - Simple repository injection vs complex ORM relations
- Multi-provider flexibility - Clean interfaces for PostgreSQL/Firestore/Mock
- Consistent error handling - Standardized error wrapping without over-abstraction

## Documentation Standards
- **Encoding**: ALWAYS create .md and documentation files using UTF-8 encoding
- **File Format**: Use plain text UTF-8, avoid binary formats for documentation

## Required Commands After Major Changes
**CRITICAL**: Always run these commands in sequence after any significant code changes:

```bash
# 1. Format Go code (must be first)
go fmt ./...

# 2. Clean up module dependencies
go mod tidy

# 3. Check for potential issues
go vet ./...

# 4. Verify compilation
go build ./...
```

## Architecture Overview

### Current Hexagonal Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    Primary Adapters                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Vanilla   │  │     Gin     │  │    Fiber    │          │
│  │    HTTP     │  │    HTTP     │  │    HTTP     │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   Application Layer                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Use Cases (40+ Entities)                   ││
│  │  Entity(17) • Event(5) • Workflow(7) • Payment(5)      ││
│  │  Product(10) • Subscription(11)                         ││
│  │  • Foreign Key Validation via Repository Injection      ││
│  └─────────────────────────────────────────────────────────┘│
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Orchestration Layer                         ││
│  │  • WorkflowEngine • StageExecutor • ActivityRunner      ││
│  │  • Human-in-the-loop support • Use case dispatch        ││
│  └─────────────────────────────────────────────────────────┘│
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Provider Management                         ││
│  │  • ProviderManager • RepositoryRegistry                 ││
│  │  • ProviderBootstrap • Registry-based DI                ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   Secondary Adapters                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ PostgreSQL  │  │  Firestore  │  │    Mock     │          │
│  │  Provider   │  │   Provider  │  │   Provider  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Firebase    │  │     JWT     │  │   No-Op     │          │
│  │    Auth     │  │    Auth     │  │    Auth     │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   AsiaPay   │  │    Gmail    │  │    Mock     │          │
│  │   Payment   │  │    Email    │  │   Services  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Domain Structure (40+ Total Entities)
The package manages **40+ distinct business entities** across **7 domains**:

- **Entity Domain (17)**: Admin, Client, ClientAttribute, Delegate, DelegateClient, Group, Location, LocationAttribute, Permission, Role, RolePermission, Staff, User, Workspace, WorkspaceUser, WorkspaceUserRole, GroupAttribute, StaffAttribute, DelegateAttribute
- **Event Domain (5)**: Event, EventAttribute, EventClient, EventProduct, EventSettings
- **Workflow Domain (7)**: Workflow, WorkflowTemplate, Stage, StageTemplate, Activity, ActivityTemplate, ActivityExecutionLog
- **Payment Domain (5)**: Payment, PaymentAttribute, PaymentMethod, PaymentProfile, PaymentProfilePaymentMethod
- **Product Domain (10)**: Collection, CollectionAttribute, CollectionPlan, CollectionParent, PriceProduct, Product, ProductAttribute, ProductCollection, ProductPlan, Resource
- **Subscription Domain (11)**: Balance, BalanceAttribute, Invoice, InvoiceAttribute, Plan, PlanAttribute, PlanSettings, PlanLocation, PricePlan, Subscription, SubscriptionAttribute

## Orchestration Layer

### Workflow Engine Architecture
The orchestration layer coordinates complex multi-step business processes:

```
┌─────────────────────────────────────────────────────────────┐
│                  WorkflowEngineService                       │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ StartWorkflowFromTemplate(template_id, input_json)      ││
│  │ ExecuteActivity(activity_id, workflow_id)               ││
│  │ AdvanceWorkflow(workflow_id)                            ││
│  │ GetWorkflowStatus(workflow_id)                          ││
│  │ ContinueWorkflow(workflow_id, activity_id, input_json)  ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Workflow Hierarchy                        │
│  WorkflowTemplate ──► Workflow (instance)                   │
│         │                    │                               │
│  StageTemplate ────► Stage (instance)                       │
│         │                    │                               │
│  ActivityTemplate ─► Activity (instance)                    │
│                              │                               │
│                    ActivityExecutionLog (audit)             │
└─────────────────────────────────────────────────────────────┘
```

### Activity Types
- **Manual**: Requires human input via `ContinueWorkflow`
- **Automated**: Executes use case code automatically
- **Approval**: Requires approval/rejection decision

### Workflow Lifecycle
1. `StartWorkflowFromTemplate` - Creates workflow instance from template
2. `GetWorkflowStatus` - Check current stage and pending activities
3. `ExecuteActivity` - Run automated activities
4. `ContinueWorkflow` - Submit human input for manual activities
5. `AdvanceWorkflow` - Move to next stage when current completes

## Use Case Implementation Patterns

### Foreign Key Validation Architecture
Use cases receive foreign key repositories via dependency injection to ensure data integrity:

**Core Implementation Steps:**
1. Input validation - Check required fields and formats
2. Business logic enrichment - Generate IDs, timestamps, default values
3. Foreign key validation - Verify referenced entities exist and are active
4. Business rule validation - Apply domain-specific constraints
5. Repository call - Execute the database operation

### Multi-Repository Dependencies
- **Simple entities** (2-3 dependencies): Direct constructor parameters
- **Complex entities** (4+ dependencies): Structured parameter grouping

## Provider System Architecture

### Provider Types Overview
| Provider Type | Purpose | Implementations |
|---------------|---------|-----------------|
| **Database** | Data persistence | PostgreSQL, Firestore, Mock |
| **Auth** | Authentication/Authorization | Firebase Auth, JWT, Mock (AllowAll) |
| **Storage** | File storage | GCS, Local, Mock |
| **ID** | Unique ID generation | Google UUID v7, NoOp |
| **Email** | Email sending | Gmail, Microsoft, Mock |
| **Payment** | Payment processing | AsiaPay, Mock |

### Environment Variables
```bash
# Database Provider
CONFIG_DATABASE_PROVIDER=firestore    # Options: firestore, postgres, mock

# Authentication Provider
CONFIG_AUTH_PROVIDER=mock_auth        # Options: firebase, jwt, mock_auth

# Storage Provider
CONFIG_STORAGE_PROVIDER=mock_storage  # Options: gcs, local, mock_storage

# ID Provider
CONFIG_ID_PROVIDER=google_uuidv7      # Options: google_uuidv7, noop, mock

# Payment Provider (optional)
CONFIG_PAYMENT_PROVIDER=asiapay       # Options: asiapay, mock
```

### Build Tags
| Build Tag | Provider | Description |
|-----------|----------|-------------|
| `firestore` | Database | Enables Firestore database provider |
| `postgres` | Database | Enables PostgreSQL database provider |
| `mock_auth` | Auth | Enables mock authorization (AllowAll) |
| `mock_storage` | Storage | Enables mock file storage |
| `google_uuidv7` | ID | Enables UUID v7 generation |
| `gmail` | Email | Enables Gmail API email provider |
| `asiapay` | Payment | Enables AsiaPay payment gateway |
| `gin` | HTTP | Enables Gin HTTP framework |
| `fiber` | HTTP | Enables Fiber HTTP framework |

**Example Build Commands:**
```bash
# Development (mock providers)
go build -tags "gin,mock_db,mock_auth,mock_storage" -o main ./cmd/server

# Production with Firestore
go build -tags "gin,firestore,firebase_auth,gcs,google_uuidv7,gmail,asiapay" -o main ./cmd/server
```

### Provider Initialization Flow
```
Container.Initialize()
    │
    ▼
providers.NewManagerWithID()
    ├── CreateDatabaseProvider(dbConfig)
    ├── CreateAuthProvider(authConfig)
    ├── CreateStorageProvider(storageConfig)
    ├── CreateIDProvider(idConfig)
    ├── CreateEmailProvider(emailConfig)
    └── CreatePaymentProvider(paymentConfig)
    │
    ▼
UseCaseInitializer.InitializeAll()
    ├── InitializeEntity(repos, services)
    ├── InitializeEvent(repos, services)
    ├── InitializePayment(repos, services)
    ├── InitializeProduct(repos, services)
    ├── InitializeSubscription(repos, services)
    └── InitializeWorkflow(repos, services)
```

## Directory Structure

```
packages/espyna/
├── cmd/server/              # Entry points
│   └── main.go
├── consumer/                # Consumer package (API surface + adapter registration)
│   ├── consumer.go          # API surface (Container, UseCases, RouteExporter)
│   ├── register.go          # Mock/noop adapter imports (always compiled)
│   ├── register_*.go        # Real adapter imports (build-tagged)
│   ├── adapter_*.go         # Adapter wrapper types
│   └── server_gin.go        # Gin utilities (build-tagged)
├── internal/
│   ├── application/
│   │   └── usecases/        # Business logic (40+ entities)
│   │       ├── entity/
│   │       ├── event/
│   │       ├── payment/
│   │       ├── product/
│   │       ├── subscription/
│   │       ├── workflow/
│   │       └── integration/
│   ├── composition/         # Dependency injection
│   │   ├── core/            # Container, initializers
│   │   ├── options/         # Configuration options
│   │   └── providers/       # Provider factories
│   │       ├── infrastructure/  # Auth, DB, Storage
│   │       └── integration/     # Email, Payment
│   ├── infrastructure/
│   │   ├── adapters/
│   │   │   ├── primary/     # HTTP handlers (Gin, Fiber, Vanilla)
│   │   │   └── secondary/   # Database, Auth, Storage implementations
│   │   │       ├── database/
│   │   │       │   ├── firestore/
│   │   │       │   ├── postgresql/
│   │   │       │   └── mock/
│   │   │       ├── auth/
│   │   │       ├── storage/
│   │   │       ├── email/
│   │   │       └── payment/
│   │   ├── providers/       # Provider implementations
│   │   └── registry/        # Repository registry
│   └── orchestration/       # Workflow engine
│       ├── engine/          # Core engine
│       ├── executor/        # Stage/activity execution
│       └── dispatcher/      # Use case dispatch
└── scripts/                 # Build and deployment scripts
```

## Testing Architecture

### Test Build Tags
```bash
go test -tags="mock_db,mock_auth" -v ./internal/application/usecases/...
```

### Test Categories
- **Success scenarios**: Basic execution and transaction handling
- **Authorization tests**: Permission validation and access control
- **Input validation**: Nil handling, required fields, length limits
- **Business logic**: Data enrichment, audit fields, domain rules
- **Error handling**: Transaction failures and system errors

## Troubleshooting Guide

### Common File Locations

**Mock Data Issues:**
```
packages/copya/data/[businessType]/[module].json
```

**Translation/Error Messages:**
```
packages/lyngua/translations/en/[businessType]/[module].json
```

**Database Providers:**
```
packages/espyna/internal/infrastructure/adapters/secondary/database/[provider]/[domain]/[module].go
```

### Common Error Patterns

| Error | Location to Check |
|-------|-------------------|
| "translation key not found" | `packages/lyngua/translations/` |
| "mock data file not found" | `packages/copya/data/` |
| "repository method not implemented" | `internal/infrastructure/adapters/secondary/database/` |
| "provider not initialized" | `internal/composition/providers/` |

## Development Guidelines

### Folder Structure Alignment
Use cases must match protobuf structure exactly:
```
esqyma/schema/v1/domain/entity/client/client.proto
    ↓
espyna/internal/application/usecases/entity/client/
```

### Use Case Standards
- Use protobuf service servers directly as interfaces
- Inject dependent repositories in constructors for foreign key validation
- Implement comprehensive validation: input, business rules, foreign keys
- Use proper error wrapping with `fmt.Errorf` and `%w` verb

### API Endpoints
Standard REST endpoints for all entity types:
- `GET /health` - Health check
- `GET /api/{domain}/{entity}` - List entities
- `POST /api/{domain}/{entity}` - Create entity
- `GET /api/{domain}/{entity}/{id}` - Get entity
- `PUT /api/{domain}/{entity}/{id}` - Update entity
- `DELETE /api/{domain}/{entity}/{id}` - Delete entity

## Architecture Benefits

- **Data Integrity** - Foreign key validation at business logic layer
- **Provider Flexibility** - Switch between providers without code changes
- **Registry Pattern** - Centralized repository management with validation
- **Multi-Framework** - Support vanilla HTTP, Gin, and Fiber simultaneously
- **Orchestration** - Complex workflow coordination with human-in-the-loop
- **Testability** - Comprehensive dependency injection for unit testing
- **Scalability** - 40+ entities across 7 domains with consistent patterns

## Notes
- **Pure Hexagonal Architecture** - No technology imports in application layer
- **40+ Total Entities** - Complete business domain coverage
- **Registry-Based DI** - No legacy factory patterns, pure provider system
- **Multi-Provider Support** - PostgreSQL, Firestore, Mock with fallback
- **Foreign Key Integrity** - Repository injection pattern ensures consistency
- **Workflow Orchestration** - Template-based workflows with stage/activity execution
