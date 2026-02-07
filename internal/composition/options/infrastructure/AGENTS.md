# options/infrastructure/

**Core system resource configuration** using Go's Functional Options pattern.

## Purpose

Configures fundamental infrastructure components that the application depends on:
- Database connections (PostgreSQL, Firestore, Mock)
- Authentication providers (Firebase, JWT, Mock)
- Storage backends (GCS, S3, Local, Mock)
- ID generation (UUID v7, NoOp)
- Server framework (Gin, Fiber, Vanilla)

## Files

| File | Providers | Options |
|------|-----------|---------|
| `options.go` | - | `ContainerOption`, setter interfaces, utilities |
| `database.go` | PostgreSQL, Firestore, Mock | `WithDatabaseFromEnv()`, `WithPostgresDatabase()` |
| `auth.go` | Firebase, JWT, Mock | `WithAuthFromEnv()`, `WithFirebaseAuth()` |
| `storage.go` | GCS, S3, Local, Mock | `WithStorageFromEnv()`, `WithGoogleCloudStorage()` |
| `id.go` | UUID v7, NoOp | `WithIDFromEnv()`, `WithGoogleUUIDv7()` |
| `server.go` | Gin, Fiber, Vanilla | `WithServerFromEnv()`, `WithGinServer()` |

## Usage

```go
import infra "leapfor.xyz/espyna/internal/composition/options/infrastructure"

container, err := core.NewContainer(
    infra.WithDatabaseFromEnv(),
    infra.WithAuthFromEnv(),
    infra.WithStorageFromEnv(),
    infra.WithServerFromEnv(),
)
```

## Environment Variables

Each provider reads from `CONFIG_*_PROVIDER` to select the implementation:
- `CONFIG_DATABASE_PROVIDER`: postgres, firestore, mock_db
- `CONFIG_AUTH_PROVIDER`: firebase_auth, jwt_auth, mock_auth
- `CONFIG_STORAGE_PROVIDER`: gcs, s3, local_storage
- `CONFIG_ID_PROVIDER`: google_uuidv7, noop
- `CONFIG_SERVER_FRAMEWORK`: gin, fiber, vanilla
