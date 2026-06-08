# registry/entityid

Single source of truth for every entity name constant and action verb constant in
espyna-golang. Every new entity adds exactly ONE constant here, not three.

## What lives here

- **Entity name constants** — one `const` value per entity (e.g. `Client = "client"`).
  The constant value is the default table/collection name.
- **Domain slices** — `var <Domain>Entities = []string{...}` groups constants per
  proto domain. `buildAll()` consolidates all slices into `All`.
- **Action verb constants** — `ActionCreate`, `ActionRead`, `ActionUpdate`,
  `ActionDelete`, `ActionList`, `ActionManage`. Used by `authcheck` and `actiongate`
  to form permission codes (`entity:action`).
- **`EntityPermission` helper** — composes an entity name + action verb into the
  permission code string consumed by the RBAC catalog.

## When to add files here

Never. There is exactly one file: `entityid.go`. All additions are edits to that file.

## Adding a new entity

1. Add one constant to the correct domain `const` block.
2. Add the constant name to the domain slice (`<Domain>Entities`).
3. The adapter, provider, use cases, and initializer each import this constant.

## Where constants are consumed

- **Adapter `init()`** — `registry.RegisterRepositoryFactory("postgresql", entityid.X, factory)`
- **DI provider** — `repoCreator.CreateRepository(entityid.X, conn, tableConfig.TableName(entityid.X))`
- **actiongate / authcheck** — `entityid.EntityPermission(entityid.Client, entityid.ActionCreate)`

## History

Entity name constants and action verb constants are the sole source of truth as of
2026-06-08. Former duplicates in `ports/security/authorization.go` were removed on
that date. Do not re-introduce action constants elsewhere.
