# usecases/service

Layer-7 use cases for service-driven domains. A service-driven domain has proto
contracts and use cases but no entityid constant, DI provider, or route config.

## Registration patterns

Two patterns — rule is callsite-driven:

1. **Typed field** — any callsite under `apps/*`. Go's `internal/` rule prevents
   apps from naming `*<X>.UseCases` as a generic type parameter. Add a field to
   `ServiceUseCases` in `usecases.go`. Exemplars: `audit`, `security`, `auth`.

2. **Dynamic registry** — ALL callsites inside `packages/espyna-golang/`. Each
   candidate's `init()` calls `service.Register`; blank imports in
   `registrar/imports.go`. One line to add a candidate; no struct edit.
   Exemplar: `tax` (caller: `RecognizeRevenueFromSubscription`).

## Sub-aggregates

| Package | Pattern | Notes |
|---------|---------|-------|
| `audit/` | typed field | Phase 1.D (20260518) |
| `security/` | typed field | Phase 1.A (20260520) |
| `auth/` | typed field | Wave 3 / Plan 2 (20260520) |
| `dashboard/` | typed field (umbrella) | Wave B — 11 per-candidate sub-fields |
| `reporting/` | typed field (umbrella) | Wave B P1.E — 5 reporting groups |
| `amortization/` | typed field | Promoted 2026-06-08 with proto contract |
| `performance/` | typed field | Performance evaluation (20260604) |
| `tax/` | dynamic registry | Caller: RecognizeRevenueFromSubscription |

## Adding a new sub-aggregate

1. Author a proto under `proto/v1/service/<X>/` in esqyma.
2. Create the package at `usecases/service/<X>/` with `usecases.go` and operation files.
3. Typed-field path: add the field to `ServiceUseCases` in `usecases.go` and wire it
   in `internal/composition/core/initializers/service/<X>.go`.
4. Dynamic path: add one blank import to `registrar/imports.go`.

Each operation file follows `Execute(ctx, *proto.Request) (*proto.Response, error)`.
Proto defines the contract; the use case wraps entity-layer logic with translation.
