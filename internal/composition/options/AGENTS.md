# options/

Go **Functional Options pattern** for composable container configuration.

## Purpose

This package contains configuration types and builder functions that define *how to configure things* - the setup and initialization of the container.

## Structure

```
options/
├── infrastructure/           # Core system resources
│   ├── options.go            # ContainerOption type, setter interfaces
│   ├── database.go           # PostgreSQL, Firestore, Mock
│   ├── auth.go               # Firebase, JWT, Mock
│   ├── storage.go            # GCS, S3, Local, Mock
│   ├── id.go                 # UUID v7, NoOp
│   └── server.go             # Gin, Fiber, Vanilla
│
└── integrations/             # External service connections
    ├── messaging/
    │   └── email.go          # Gmail, Microsoft, Mock
    └── payment/
        └── payment.go        # Stripe, AsiaPay, Mock
```

## When to import

```go
// Infrastructure options
import infra "leapfor.xyz/espyna/internal/composition/options/infrastructure"

// Integration options
import "leapfor.xyz/espyna/internal/composition/options/integrations/messaging"
import "leapfor.xyz/espyna/internal/composition/options/integrations/payment"
```

## Usage Example

```go
container, err := core.NewContainer(
    // Infrastructure
    infra.WithDatabaseFromEnv(),
    infra.WithAuthFromEnv(),
    infra.WithStorageFromEnv(),
    infra.WithServerFromEnv(),
)

// Integrations
messaging.WithEmailFromEnv()(container)
payment.WithPaymentFromEnv()(container)
```

## Not to be confused with

**`contracts/`** - Contains DDD-style interfaces that define *behavioral boundaries*. That package defines `Service`, `UseCase`, `Repository` interfaces, not configuration.
