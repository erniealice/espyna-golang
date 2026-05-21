# usecases/

**Application Use Cases** - implement business logic as single-purpose operations. Each use case encapsulates one business action (create, read, update, delete, list, or custom operations) with validation, enrichment, and transaction support.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         COMPOSITION LAYER                                    │
│                    composition/routing/handlers/                             │
│                      (HTTP handlers call use cases)                         │
└─────────────────────────────────────────┬───────────────────────────────────┘
                                          │ calls Execute()
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            USE CASES LAYER                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         UseCase.Execute()                            │   │
│  │                                                                      │   │
│  │  1. Input Validation      → Nil checks, required fields              │   │
│  │  2. Business Validation   → validateBusinessRules()                  │   │
│  │  3. Business Enrichment   → applyBusinessLogic()                     │   │
│  │  4. Transaction Wrapper   → TransactionService.ExecuteInTransaction  │   │
│  │  5. Core Execution        → executeCore() → Repository calls         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                      │
│         ┌────────────────────────────┼────────────────────────────┐        │
│         ▼                            ▼                            ▼        │
│  ┌─────────────┐           ┌─────────────────┐           ┌─────────────┐  │
│  │Repositories │           │    Services     │           │   Ports     │  │
│  │ (via proto) │           │  (injected DI)  │           │ (interfaces)│  │
│  └─────────────┘           └─────────────────┘           └─────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INFRASTRUCTURE LAYER                                 │
│              adapters/secondary/database/{provider}/{domain}/               │
│                         (Repository implementations)                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

After Plan 2 (`20260522-usecases-realignment`, landed 2026-05-21), entity-layer use cases
live under `domain/` and service-layer use cases live under `service/`:

```
usecases/
├── auth/                       # Identity-lifecycle use cases (stays here; no proto/v1/domain/auth)
│   ├── authenticate_session.go # Cookie/token → Identity
│   ├── issue_session.go        # Mint a new session token
│   └── invalidate_session.go   # Logout / session termination
│
├── domain/                     # Entity-layer use cases (mirrors proto/v1/domain/<X>/)
│   ├── common/                 # Cross-domain use cases
│   │   └── attribute/          # Generic attribute CRUD
│   │
│   ├── entity/                 # Entity domain (17+ entities)
│   │   ├── admin/
│   │   ├── client/
│   │   ├── workspace/
│   │   ├── workspace_user/
│   │   └── workspace_user_role/
│   │   └── ...
│   │
│   ├── event/                  # Event domain
│   ├── product/                # Product domain
│   ├── subscription/           # Subscription domain
│   ├── revenue/                # Revenue domain
│   ├── expenditure/            # Expenditure domain
│   ├── treasury/               # Treasury domain
│   ├── ledger/                 # Ledger domain
│   ├── document/               # Document sub-aggregate (wraps attachment + template)
│   ├── operation/              # Operation domain
│   ├── workflow/               # Workflow domain
│   └── ...                     # All other proto/v1/domain/ domains
│
└── service/                    # Service-layer use cases (mirrors proto/v1/service/<X>/)
    ├── audit/
    ├── auth/                   # Auth application service (orchestrates Session + User)
    ├── dashboard/
    ├── reporting/
    ├── registrar/              # Cycle-break blank-import registrar (formerly serviceregistrar/)
    ├── security/
    └── tax/
```

## Use Case File Pattern

Each entity folder contains these files:

```
domain/{domain}/{name}/
├── usecases.go                 # Aggregates all use cases + NewUseCases()
├── create_{name}.go            # Create use case
├── read_{name}.go              # Read (get by ID) use case
├── update_{name}.go            # Update use case
├── delete_{name}.go            # Delete use case
├── list_{name}s.go             # List with pagination/filters
├── get_{name}_list_page_data.go   # Frontend list page data
├── get_{name}_item_page_data.go   # Frontend item page data
├── find_or_create_{name}.go    # (optional) Upsert pattern
├── *_test.go                   # Unit tests
└── ... (custom use cases)
```

## Use Case Structure

Each use case follows this consistent pattern:

```go
package client

// 1. Repository dependencies (proto-generated interfaces)
type CreateClientRepositories struct {
    Client clientpb.ClientDomainServiceServer
    User   userpb.UserDomainServiceServer  // Cross-entity dependency
}

// 2. Service dependencies (injected via DI)
type CreateClientServices struct {
    AuthorizationService ports.AuthorizationService
    TransactionService   ports.TransactionService
    TranslationService   ports.TranslationService
    IDService            ports.IDService
}

// 3. Use case struct
type CreateClientUseCase struct {
    repositories CreateClientRepositories
    services     CreateClientServices
}

// 4. Constructor
func NewCreateClientUseCase(
    repositories CreateClientRepositories,
    services CreateClientServices,
) *CreateClientUseCase

// 5. Execute method (main entry point)
func (uc *CreateClientUseCase) Execute(
    ctx context.Context,
    req *clientpb.CreateClientRequest,
) (*clientpb.CreateClientResponse, error)
```

## Execution Flow

```
Execute(ctx, req)
       │
       ▼
┌──────────────────────┐
│  1. Input Validation │ → nil checks, required request
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  2. Business Rules   │ → validateBusinessRules()
│     Validation       │   - Required fields
└──────────┬───────────┘   - Format validation (email, etc.)
           │               - Length constraints
           ▼               - Business constraints
┌──────────────────────┐
│  3. Business Logic   │ → applyBusinessLogic()
│     Enrichment       │   - Generate IDs (via IDService)
└──────────┬───────────┘   - Set audit fields (dates)
           │               - Set default values
           ▼               - Normalize data
┌──────────────────────┐
│  4. Transaction?     │ → if TransactionService.SupportsTransactions()
│                      │     ExecuteInTransaction(ctx, func)
└──────────┬───────────┘   else direct call
           │
           ▼
┌──────────────────────┐
│  5. Core Execution   │ → executeCore()
│                      │   - Cross-entity operations
└──────────┬───────────┘   - Repository.Create/Read/Update/Delete
           │
           ▼
      Response/Error
```

## Service Dependencies

| Service | Purpose | Interface |
|---------|---------|-----------|
| AuthorizationService | Permission checks | `ports.AuthorizationService` |
| TransactionService | Atomic operations | `ports.TransactionService` |
| TranslationService | i18n error messages | `ports.TranslationService` |
| IDService | Generate UUIDs | `ports.IDService` |

## Domain Summary

| Domain | Entities | Description |
|--------|----------|-------------|
| auth | 3 | Identity-lifecycle: authenticate_session, issue_session, invalidate_session (bypasses authcheck — see invariant below) |
| entity | 17 | Users, clients, staff, roles, permissions, workspaces |
| event | 3 | Events, event clients, event attributes |
| payment | 4 | Payments, methods, profiles |
| product | 9 | Products, collections, pricing |
| subscription | 8 | Plans, subscriptions, invoices, balances |
| workflow | 6 | Workflows, stages, activities, templates |
| common | 1 | Shared attribute system |
| integration | 2 | Email, payment gateway operations |

### `auth/` invariant

`auth/` is the only domain whose use cases are exempt from `authcheck.Check`
(see `authcheck_coverage_test.go` `skipDirs`). A file belongs under `auth/` only
if it (a) establishes identity — login, authenticate_session, issue_session,
register, password reset — or (b) terminates a previously established session
(invalidate_session, rotate_session). Authenticated business actions that
happen to touch auth entities (e.g. "admin revokes another user's sessions",
"user lists their own active sessions") belong in `entity/session/` with
`authcheck.Check` wired in. Treat `auth/` as a semantic boundary, not a
convenience bucket — widening it defeats the coverage test.

## Aggregated UseCases Pattern

Each entity has a `usecases.go` that aggregates all use cases:

```go
// In entity/client/usecases.go

type UseCases struct {
    CreateClient          *CreateClientUseCase
    ReadClient            *ReadClientUseCase
    UpdateClient          *UpdateClientUseCase
    DeleteClient          *DeleteClientUseCase
    ListClients           *ListClientsUseCase
    GetClientListPageData *GetClientListPageDataUseCase
    GetClientItemPageData *GetClientItemPageDataUseCase
    FindOrCreateClient    *FindOrCreateClientUseCase
    GetClientByEmail      *GetClientByEmailUseCase
}

func NewUseCases(
    repositories ClientRepositories,
    services ClientServices,
) *UseCases
```

## Import Patterns

```go
// Use case package
import "leapfor.xyz/espyna/internal/application/usecases/domain/entity/client"

// Create aggregated use cases
clientUseCases := client.NewUseCases(repos, services)

// Call specific use case
resp, err := clientUseCases.CreateClient.Execute(ctx, req)
```

## Translation Support

Use cases support i18n for error messages:

```go
// In validateBusinessRules()
if client.User.EmailAddress == "" {
    return errors.New(contextutil.GetTranslatedMessageWithContext(
        ctx,
        uc.services.TranslationService,
        "client.validation.email_required",      // translation key
        "Email address is required [DEFAULT]",   // fallback
    ))
}
```

## Testing Pattern

Each use case has corresponding tests:

```go
// create_client_test.go
func TestCreateClientUseCase_Execute(t *testing.T) {
    // Setup mock repositories
    mockClientRepo := &mockClientRepo{}
    mockUserRepo := &mockUserRepo{}

    repos := CreateClientRepositories{Client: mockClientRepo, User: mockUserRepo}
    services := CreateClientServices{
        TransactionService: ports.NewNoOpTransactionService(),
        TranslationService: ports.NewNoOpTranslationService(),
        IDService:          ports.NewNoOpIDService(),
    }

    uc := NewCreateClientUseCase(repos, services)

    // Test execution
    resp, err := uc.Execute(ctx, req)
    // Assertions...
}
```

## Related Packages

| Package | Purpose |
|---------|---------|
| `application/ports/` | Service interfaces (AuthorizationService, etc.) |
| `composition/core/initializers/domain/` | Entity-layer use case initialization per domain |
| `composition/core/initializers/service/` | Service-layer use case initialization |
| `composition/routing/handlers/` | HTTP handlers that call use cases |
| `esqyma/pkg/schema/v1/domain/` | Proto-generated request/response types |

## Key Design Decisions

1. **One file per use case** - Clear separation, easy to find and test
2. **Grouped dependencies** - Repositories and Services structs, not individual params
3. **Transaction wrapper** - Optional transaction support via TransactionService
4. **Translation support** - All user-facing errors go through TranslationService
5. **ID generation** - Centralized via IDService, not scattered UUID calls
6. **Proto-based contracts** - Repository interfaces from protobuf, not hand-written
7. **Page data use cases** - Dedicated use cases for frontend list/item pages
8. **NoOp services** - Default no-op implementations for optional services
