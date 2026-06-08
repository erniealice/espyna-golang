# internal/application/ports

The port layer of espyna-golang's hexagonal architecture. Ports define the
contracts that separate the application core from infrastructure details.

## The proto-is-the-port principle

Most request/response contracts in this system are proto services. A hand-written
Go port interface is justified **only** when the contract requires Go runtime
mechanics that proto cannot express.

The test: "Can this interface be expressed as a proto RPC with `context.Context` as
the only Go addition?" If yes — use the proto-generated interface. No port needed.

## What qualifies as a genuine port

| Port | Reason proto cannot express it |
|------|--------------------------------|
| `Transactor` | Closure passed to `ExecuteInTransaction` |
| `IDGenerator` | Side-effecting random generation; no wire shape |
| `Translator` | Variadic `params ...any` — not expressible in proto3 |
| `DatabaseProvider` | `*sql.DB` lifecycle — Go-specific connection pool |
| `ServerProvider` | `net/http` lifecycle — Go-specific |
| `StorageProvider` / `StreamingStorageProvider` | `io.Reader` / `io.ReadCloser` streaming |
| `MigrationService` | Filesystem + DDL concerns; no proto equivalent |

## Subdirectories

- `infrastructure/` — Runtime mechanics: transactions, ID generation, DB/server lifecycle,
  storage streaming, migrations.
- `security/` — Authorization port (transitioning to `proto/v1/service/security/`).
- `domain/` — Domain-capability ports. Several already return proto types — migration
  candidates.
- `integration/` — Third-party provider lifecycle contracts; message shapes migrating to
  proto.

## When to add a port here

Only when the four-signal test fails. A port whose every method is
`(ctx, *SomeProtoRequest) (*SomeProtoResponse, error)` belongs in
`proto/v1/service/<X>/` as a Layer-7 use case, not here.
