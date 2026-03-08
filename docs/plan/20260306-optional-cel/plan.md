# Optional CEL Workflow Engine — Design Plan

**Date:** 2026-03-06
**Branch:** `dev/20260306-optional-cel`
**Status:** Draft
**Package:** espyna-golang-ryta

---

## Overview

Make the CEL expression engine (`github.com/google/cel-go`) an **optional build-tag dependency** in espyna so that consumer apps like `service-admin` that don't use workflow orchestration can drop 5 transitive dependencies (~4MB of compiled binary) from their builds.

---

## Motivation

CEL brings 5 transitive dependencies into every consumer app:

| Dep | Size impact |
|-----|-------------|
| `github.com/google/cel-go` | Core engine |
| `cel.dev/expr` | AST definitions |
| `github.com/antlr4-go/antlr/v4` | Parser runtime |
| `github.com/stoewer/go-strcase` | String utils |
| `golang.org/x/exp` | Experimental stdlib |

`service-admin` doesn't use workflow orchestration at all — it only manages workflow templates (CRUD). Yet it pays the dependency cost because the CEL evaluator is compiled unconditionally.

---

## Architecture

**Approach: Build tag `cel` + interface abstraction**

The strategy follows espyna's existing pattern (see `//go:build jwt_auth`, `//go:build asiapay`, etc.) — isolate CEL behind a build tag and provide a no-op stub when the tag is absent.

```
engine/
  condition_evaluator.go          ← NEW: interface definition (no build tag)
  cel_evaluator.go                ← MODIFY: add //go:build cel
  cel_evaluator_noop.go           ← NEW: //go:build !cel — returns nil evaluator
  execute_activity.go             ← MODIFY: accept interface, not concrete *CELEvaluator
```

**Interface:**
```go
// ConditionEvaluator evaluates workflow condition expressions.
// nil is a valid "no-op" evaluator — callers must nil-check.
type ConditionEvaluator interface {
    EvaluateCondition(expression string, context map[string]any) (bool, error)
}
```

**Config change:** `CONFIG_WORKFLOW_ENGINE_MODE` gains a new value `none` that skips `initializeWorkflowEngine()` entirely. This is the cleanest opt-out for apps that don't need any workflow features.

---

## Implementation Steps

### Phase 1: Build-tag split for CEL evaluator

1. Create `condition_evaluator.go` — define `ConditionEvaluator` interface (no build tag)
2. Add `//go:build cel` to existing `cel_evaluator.go`
3. Create `cel_evaluator_noop.go` with `//go:build !cel` — `NewCELEvaluator()` returns `nil, nil`
4. Update `execute_activity.go:23` — change field type from `*CELEvaluator` to `ConditionEvaluator`
5. Update `execute_activity.go:28` — `NewCELEvaluator()` call is unchanged (both build variants provide it)

### Phase 2: Add `none` mode to workflow engine lifecycle

6. Add `ModeNone` to `contracts/lifecycle.go`
7. Update `lifecycle.go:IsValid()` to include `ModeNone`
8. Add `case orchcontracts.ModeNone:` to `container.go:386` — skip `initializeWorkflowEngine()` entirely, log "⏭️ Workflow Engine disabled (none mode)"
9. Update `.env.example` line 7 comment: `# none | late | eager | lazy`

### Phase 3: Update consumer apps

10. Update `apps/service-admin/.env` — set `CONFIG_WORKFLOW_ENGINE_MODE=none`
11. Remove `cel` from service-admin build tags (it shouldn't have it)
12. Run `go mod tidy` in `apps/service-admin/` — verify CEL deps drop from go.mod
13. Verify build: `go build -tags "google_uuidv7,mock_auth,mock_storage,noop,postgresql,vanilla"`

---

## File References

| File | Change | Phase |
|------|--------|-------|
| `packages/espyna-golang-ryta/internal/orchestration/engine/condition_evaluator.go` | **New file** — `ConditionEvaluator` interface | 1 |
| `packages/espyna-golang-ryta/internal/orchestration/engine/cel_evaluator.go` | Add `//go:build cel` tag | 1 |
| `packages/espyna-golang-ryta/internal/orchestration/engine/cel_evaluator_noop.go` | **New file** — noop stub with `//go:build !cel` | 1 |
| `packages/espyna-golang-ryta/internal/orchestration/engine/execute_activity.go` | Change `*CELEvaluator` → `ConditionEvaluator` at lines 23, 27 | 1 |
| `packages/espyna-golang-ryta/internal/orchestration/contracts/lifecycle.go` | Add `ModeNone` constant, update `IsValid()` | 2 |
| `packages/espyna-golang-ryta/internal/composition/core/container.go` | Add `ModeNone` case at line 386 | 2 |
| `packages/espyna-golang-ryta/.env.example` | Update comment on line 7 | 2 |
| `apps/service-admin/.env` | Set `CONFIG_WORKFLOW_ENGINE_MODE=none` | 3 |

---

## Context & Sub-Agent Strategy

**Estimated files to read:** 8
**Estimated files to modify:** 6 (+ 2 new)
**Estimated context usage:** Low (<30 files)

No sub-agents needed. Single session is sufficient.

---

## Risk & Dependencies

| Risk | Impact | Mitigation |
|------|--------|------------|
| Apps that use CEL forget to add build tag | Workflow conditions silently skip (fail-open) | `NewCELEvaluator()` noop returns nil — existing nil-check at `execute_activity.go:96` already handles this gracefully |
| `go mod tidy` doesn't drop deps | Low — deps remain as indirect | Run tidy inside service-admin dir after setting mode to none; verify with `go list -m all` |

**Dependencies:**
- Phase 2 depends on Phase 1 (ModeNone uses the noop evaluator path)
- Phase 3 depends on Phase 2 (service-admin needs the new mode value)

---

## Acceptance Criteria

- [ ] `go build` with `cel` tag: CEL evaluator works as before (no regression)
- [ ] `go build` without `cel` tag: compiles, `NewCELEvaluator()` returns nil, conditions are skipped
- [ ] `CONFIG_WORKFLOW_ENGINE_MODE=none` skips engine initialization entirely
- [ ] `apps/service-admin/go.mod` no longer contains cel-go, antlr4, cel.dev/expr, go-strcase, x/exp after `go mod tidy`
- [ ] `apps/service-admin` builds and runs correctly with mode=none

---

## Design Decisions

**Why build tag + ModeNone, not just ModeNone?**

`ModeNone` alone would skip engine *initialization* at runtime, but the CEL import in `cel_evaluator.go` still forces the Go compiler to link the CEL packages. Build tags are the only way to truly eliminate the dependency from the binary and go.mod. The two mechanisms complement each other: build tag controls *compilation*, ModeNone controls *runtime initialization*.

**Why not a separate Go module for the engine?**

That would require `packages/espyna-golang-ryta/orchestration/engine/go.mod` — splitting espyna into sub-modules. While clean, it's a much larger refactor and breaks the current monorepo module graph. Build tags achieve the same compile-time isolation with zero structural changes.
