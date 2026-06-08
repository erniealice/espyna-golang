# ports/domain

Domain-capability ports. Several interfaces here already take and return proto
types — they are candidates for promotion to proto services. This is a
transitional package; it shrinks as each interface graduates.

## Residents

| Interface | Status | Note |
|-----------|--------|------|
| `Translator` | **Stays** | Variadic `params ...any` cannot be expressed in proto3. Genuine port. |
| `LedgerReportingService` | **Migrating** | All methods take/return proto types. Each method is being promoted to `proto/v1/service/reporting/<group>/` in Wave B P1.E.1-5. The fat interface shrinks one method group per sub-commit. |
| `OutcomeEvaluationService` | **Migrating** | All methods take/return proto types — signal that it should become a proto service under `proto/v1/service/`. |
| `WorkflowEngineService` | **Migrating** | RPC-shaped methods with proto request/response → should become proto services. |
| `ActivityExecutor` | **Stays** | Takes `map[string]any` callback — not expressible in proto without losing the dynamic dispatch contract. |
| `ExecutorRegistry` | **Stays** | Dynamic lookup by code string returning a Go interface — composition concern, not a wire contract. |

## When to add a file here

Before adding a new interface here, ask: "Do all methods on this interface take a
proto request and return a proto response?" If yes, it belongs in
`proto/v1/service/<X>/` as a service-driven domain with Layer-7 use cases. Only
interfaces with genuine Go-mechanic requirements (variadic args, `map[string]any`
callbacks, function closures) belong here.

See `ports/README.md` for the full four-signal test.
