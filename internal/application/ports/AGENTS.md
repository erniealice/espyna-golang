# Espyna Ports Layer - Agent Guide

## Package Overview

The **ports** package defines the **behavioral contracts** for the espyna application's hexagonal architecture. This package contains Go interfaces that define HOW external systems and infrastructure services should behave, independent of their implementation details.

## Directory Structure

```
ports/
├── AGENTS.md                   # This file
├── common.go                   # Shared types (ApplicationError)
├── exports.go                  # Re-exports for backward compatibility
│
├── infrastructure/             # Core infrastructure ports
│   ├── database.go             # DatabaseProvider, RepositoryProvider
│   ├── database_config.go      # DatabaseConfigAdapter
│   ├── auth.go                 # AuthProvider, AuthService
│   ├── auth_config.go          # AuthConfigAdapter
│   ├── storage.go              # StorageProvider, StorageError
│   ├── storage_config.go       # StorageConfigAdapter
│   ├── id.go                   # IDService
│   ├── transaction.go          # TransactionService
│   └── migration.go            # MigrationService
│
├── integration/                # External service integration ports
│   ├── email.go                # EmailProvider, EmailMessage
│   └── payment.go              # PaymentProvider, CheckoutSessionParams
│
├── domain/                     # Domain-specific ports
│   ├── workflow.go             # WorkflowEngineService, ExecutorRegistry
│   └── translation.go          # TranslationService
│
└── security/                   # Security and authorization ports
    ├── authorization.go        # AuthorizationService, AuthorizationProvider
    └── errors.go               # AuthorizationError, error codes
```

## Importing Ports

### Backward Compatible Import (Recommended for existing code)
```go
import "leapfor.xyz/espyna/internal/application/ports"

// Use types directly
var db ports.DatabaseProvider
var email ports.EmailProvider
```

### Sub-package Import (Recommended for new code)
```go
import (
    "leapfor.xyz/espyna/internal/application/ports/infrastructure"
    "leapfor.xyz/espyna/internal/application/ports/integration"
    "leapfor.xyz/espyna/internal/application/ports/domain"
    "leapfor.xyz/espyna/internal/application/ports/security"
)

// Use types from specific packages
var db infrastructure.DatabaseProvider
var email integration.EmailProvider
var authz security.AuthorizationService
```

## Critical Distinction: Ports vs Proto Contracts

### **Ports (This Package)**
- **What**: Go interfaces defining adapter behavior
- **Purpose**: Hexagonal architecture boundaries, dependency inversion
- **Scope**: Internal application architecture (Go only)
- **Location**: `packages/espyna/internal/application/ports/`
- **When to use**: Define contracts for infrastructure adapters

### **Proto Contracts (Esqyma Package)**
- **What**: Protobuf schemas defining data structures and gRPC services
- **Purpose**: Data serialization, network transport, cross-language compatibility
- **Scope**: Inter-service communication, API contracts
- **Location**: `packages/esqyma/schema/v1/`
- **When to use**: Define data shapes and gRPC service endpoints

---

## Directory Responsibilities

### `infrastructure/`
Core system resources that the application depends on:

| File | Purpose |
|------|---------|
| `database.go` | Database connection and repository providers |
| `auth.go` | Authentication provider and service interfaces |
| `storage.go` | Object storage (S3, GCS, Local) providers |
| `id.go` | ID generation service (UUID, etc.) |
| `transaction.go` | Database transaction management |
| `migration.go` | Database schema migration |

### `integration/`
External service integrations:

| File | Purpose |
|------|---------|
| `email.go` | Email provider (Gmail, SendGrid, SMTP) |
| `payment.go` | Payment provider (Stripe, AsiaPay, etc.) |

### `domain/`
Domain-specific service contracts:

| File | Purpose |
|------|---------|
| `workflow.go` | Workflow orchestration engine |
| `translation.go` | Translation/localization service |

### `security/`
Security and authorization:

| File | Purpose |
|------|---------|
| `authorization.go` | Permission checking, role management |
| `errors.go` | Authorization error types and codes |

---

## Design Principles

### **1. Dependency Inversion Principle**
Use cases depend on port interfaces (abstractions), not concrete implementations:
```go
// Good: Use case depends on interface
type SendEmailUseCase struct {
    emailProvider ports.EmailProvider  // ← Interface (port)
}

// Bad: Use case depends on concrete implementation
type SendEmailUseCase struct {
    emailProvider *sendgrid.Adapter  // ❌ Concrete implementation
}
```

### **2. Interface Segregation**
Interfaces should be focused and specific:
```go
// Good: Focused interfaces
type EmailSender interface {
    SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)
}

type EmailReader interface {
    GetInboxMessages(ctx context.Context, req *pb.GetInboxMessagesRequest) (*pb.GetInboxMessagesResponse, error)
}
```

### **3. Use Proto Types, Define Behavior**
Ports should use proto-generated types but define behavior:
```go
import pb "leapfor.xyz/esqyma/golang/v1/integration/email"

type EmailProvider interface {
    // Uses proto request/response types ✅
    SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)

    // Defines lifecycle behavior (NOT in proto) ✅
    Initialize(config *pb.EmailProviderConfig) error
    Close() error
    IsHealthy(ctx context.Context) error
}
```

### **4. Context-Aware**
All I/O operations should accept `context.Context`:
```go
// Good: Context-aware
SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)

// Bad: No context
SendEmail(req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)
```

---

## Common Patterns

### **Pattern 1: Provider Lifecycle**
All infrastructure providers follow this pattern:
```go
type XxxProvider interface {
    Name() string                          // Provider identification
    Initialize(config *pb.Config) error    // Setup with proto config
    IsHealthy(ctx context.Context) error   // Health check
    Close() error                          // Cleanup
    IsEnabled() bool                       // Feature flag
}
```

### **Pattern 2: Request/Response Methods**
Methods that perform operations use proto request/response pairs:
```go
type EmailProvider interface {
    SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)
    GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.GetMessageResponse, error)
}
```

### **Pattern 3: NoOp Fallbacks**
Each service interface has a NoOp implementation for testing/fallback:
```go
// Create fallback services
idService := ports.NewNoOpIDService()
txService := ports.NewNoOpTransactionService()
authzService := ports.NewNoOpAuthorizationService()
translationService := ports.NewNoOpTranslationService()
```

---

## Adding New Ports

### Adding a new infrastructure port:
1. Create file in `infrastructure/` directory
2. Define interface with lifecycle methods
3. Add type alias to `exports.go` for backward compatibility
4. Update this AGENTS.md

### Adding a new integration port:
1. Create file in `integration/` directory
2. Define interface with proto request/response methods
3. Add type alias to `exports.go`
4. Update this AGENTS.md

### Adding a new domain port:
1. Create file in `domain/` directory
2. Define interface with domain-specific methods
3. Add type alias to `exports.go`
4. Update this AGENTS.md

---

## Testing Strategy

### **Mock Implementations**
Ports enable easy mocking for testing:
```go
type MockEmailProvider struct {
    SendEmailFunc func(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error)
}

func (m *MockEmailProvider) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
    if m.SendEmailFunc != nil {
        return m.SendEmailFunc(ctx, req)
    }
    return &pb.SendEmailResponse{Success: true}, nil
}
```

### **Test Adapters**
Build test-specific adapters:
```go
type InMemoryEmailProvider struct {
    sentEmails []*pb.EmailMessage
}

func (p *InMemoryEmailProvider) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
    message := convertRequestToMessage(req)
    p.sentEmails = append(p.sentEmails, message)
    return &pb.SendEmailResponse{Success: true, MessageId: "test-123"}, nil
}
```

---

## Key Takeaways

1. **Ports = Behavior (Go Interfaces)**, Proto = Data + Services (Network RPC)
2. **Ports enable dependency inversion**, Proto enables serialization and cross-service communication
3. **Ports use proto types**, but define additional behavior (lifecycle, health, capabilities)
4. **Proto services generate gRPC servers**, Port interfaces are implemented by adapters
5. **Both coexist**: gRPC servers can call use cases, which depend on port interfaces
6. **Backward compatibility**: Use `exports.go` to re-export types to the root `ports` package

---

## Further Reading

- Hexagonal Architecture: https://alistair.cockburn.us/hexagonal-architecture/
- Dependency Inversion Principle: SOLID principles
- gRPC Services: https://grpc.io/docs/what-is-grpc/core-concepts/
- Protocol Buffers: https://protobuf.dev/
