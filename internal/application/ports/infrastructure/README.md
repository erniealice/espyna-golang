# ports/infrastructure

Genuine runtime ports that require Go-specific mechanics proto cannot express.
These interfaces form the driven-port boundary between the application core and
the infrastructure layer below it.

## Residents and why they are ports

| Interface | Go mechanic that justifies it |
|-----------|-------------------------------|
| `Transactor` | Closure-based tx: `ExecuteInTransaction(ctx, func(ctx) error)`. Proto has no closure type. |
| `IDGenerator` | Side-effecting random generation. No request/response shape. |
| `DatabaseProvider` | `*sql.DB` lifecycle — `GetConnection() any` returns a Go driver type. |
| `ServerProvider` | `net/http.Server` lifecycle — Go-specific start/stop. |
| `StorageProvider` | Core storage operations using proto request/response types. |
| `StreamingStorageProvider` | `io.Reader` upload, `io.ReadCloser` download — bounded-memory streaming that proto bytes fields cannot model without full buffering. |
| `MigrationService` | Filesystem scanning and DDL execution — no proto equivalent. |
| `PoolSizer` | Optional `MaxConns() int` extension for concurrency-aware callers. |

## When to add a file here

Only when you need a new interface that requires one of the mechanics above.
If the new interface takes a proto request and returns a proto response, it belongs
in `proto/v1/service/<X>/` as a service-driven domain, not here.

## Implemented by

Adapters under `contrib/postgres/`, `contrib/aws/`, `contrib/azure/`,
`contrib/google/`, `contrib/fiber/`, and the mock infrastructure layer.
Each adapter self-registers via `init()` and is blank-imported at startup.
