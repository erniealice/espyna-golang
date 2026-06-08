# ports/security

Authorization port — the driven boundary between the application's permission
checks and the RBAC infrastructure that resolves them.

## Current residents

- `Authorizer` — checks whether a principal has a permission code. Used by
  `authcheck.Check` at the top of every use case `Execute` method.
- `AuthorizationProvider` — lifecycle contract for auth providers
  (initialize, close). Will move to `ports/infrastructure/` when the broader
  provider-lifecycle consolidation lands.
- `NoOpAuthorizer` — always-allow fallback for tests and pre-login pages.

## Transition status (as of 2026-06-08)

`GetUserPermissionCodes` — the query that returns all effective permission codes
for a principal — has been promoted to a proto service and a Layer-7 use case:

- Proto: `proto/v1/service/security/permission_query.proto`
- Use case: `usecases/service/security/get_user_permission_codes.go`
- Wiring: `internal/composition/core/initializers/service/security.go`

The `Authorizer` interface itself remains here as the runtime check port used
inside use cases. The entity/action constants that were previously duplicated in
this package were removed 2026-06-08; the single source of truth is now
`registry/entityid/entityid.go`.

## When to add a file here

Do not add new interfaces here speculatively. If you need a new authorization
concept, evaluate whether it is better expressed as a proto service under
`proto/v1/service/security/`. A hand-written Go port is warranted only when
the contract cannot be a proto RPC (see `ports/README.md` for the test).
