# Optional CEL Workflow Engine — Progress Log

**Plan:** [plan.md](./plan.md)
**Started:** 2026-03-06
**Branch:** `dev/20260306-optional-cel`

---

## Phase 1: Build-tag split for CEL evaluator — COMPLETE

- [x] Create `condition_evaluator.go` — ConditionEvaluator interface
- [x] Add `//go:build cel` to `cel_evaluator.go`
- [x] Create `cel_evaluator_noop.go` with `//go:build !cel`
- [x] Update `execute_activity.go` — use ConditionEvaluator interface

---

## Phase 2: Add `none` mode to workflow engine lifecycle — COMPLETE

- [x] Add `ModeNone` to `contracts/lifecycle.go`
- [x] Add `ModeNone` case to `container.go`
- [x] Update `.env.example` comment and default

---

## Phase 3: Update consumer apps — COMPLETE

- [x] Set `CONFIG_WORKFLOW_ENGINE_MODE=none` in service-admin (.env + .env.alpha)
- [x] Run `go mod tidy` in service-admin
- [x] Verify local build succeeds (google_uuidv7,mock_auth,mock_storage,noop,postgresql,vanilla,lyngua)
- [x] Verify alpha build succeeds (build-alpha.ps1 — gcp_storage,google,mock_auth,noop,postgresql,vanilla)
- [x] CEL deps remain in go.mod (expected — go mod tidy considers all build tag permutations) but are NOT compiled into binary

---

## Summary

- **Phases complete:** 3 / 3
- **Files modified:** 4 modified + 2 created = 6 total

---

## Note on go.mod deps

`go mod tidy` does NOT remove CEL deps from go.mod because it considers ALL possible build tag combinations. Since `cel_evaluator.go` (with `//go:build cel`) is reachable via the espyna module graph, tidy keeps the deps for completeness. However, the deps are **not compiled into the binary** when built without the `cel` tag. To fully remove them from go.mod, the CEL code would need to live in a separate Go sub-module — a larger refactor deferred for now.

---

## Skipped / Deferred

| Item | Reason |
|------|--------|
| Remove CEL deps from go.mod | Would require separate Go sub-module for engine; deferred |
