# internal/application/shared

Pure leaf packages with charters. Every package here must be admitted via the
Rule-of-Three (3+ cross-domain consumers, or 2 with explicit reviewer override for
an unambiguously pure leaf utility).

## Admission rules

- Admitted only when the package serves 3+ consumers across different domains.
- Must not import proto entity types, DB drivers, adapter packages, or anything
  under `internal/application/usecases/`.
- Must declare a charter as the package-level doc-comment: what it does, what it
  must not import, and a current consumers list.
- If a package here starts wanting proto types, it is a service-driven domain in
  disguise — escalate to `proto/v1/service/<X>/` instead.

## Current residents

| Package | Purpose | Gate |
|---------|---------|------|
| `actiongate/` | Gate 1 — RBAC capability checks: "can this principal perform this action on this entity?" | Gate 1 |
| `resourcegate/` | Gate 2 — junction-based membership checks: "can this principal access resources scoped to this client/subscription?" | Gate 2 |
| `authcheck/` | Deprecated entry point — calls `actiongate` under the hood. New callers should use `actiongate` directly. | — |
| `amortize_schedule/` | Pure period and tranche math engine. No proto, no DB. The `usecases/service/amortization/` wrapper provides the versioned proto contract. | — |
| `context/` | Principal ID extraction from `context.Context`. | — |
| `listdata/` | Go helper layer over `proto/v1/domain/common/{pagination,sort,filter}`. | — |
| `testutil/` | Test infrastructure helpers. | — |
| `evaluation_score/` | Weighted-average score computation over snapshotted evaluation responses. Pure math, no proto, no DB. | — |

## When to add a package here

1. Run the Rule-of-Three test: does this utility have 3+ consumers across different
   domains? (Or 2 unambiguously independent domains — document the override.)
2. Verify the four-signal test from `hexagonal-rules.md` §3: if any signal fires
   (wire shape, multi-language, versioned API, multi-implementation), escalate to
   `proto/v1/service/<X>/` instead.
3. Write a charter doc-comment with MUST NOT import list and consumers list.
