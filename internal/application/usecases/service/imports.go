// Package service imports.go — DEPRECATED LOCATION for the central
// blank-import file for dynamically-registered service-driven candidate
// packages.
//
// **Why this file no longer hosts the blank imports:** the registry
// itself (Register/Get/Deps) lives in `package service` (see
// usecases.go), and every dynamically-registered child package MUST
// import `package service` to call [service.Register]. If this file
// (in `package service`) then blank-imports those child packages, Go
// detects the cycle and refuses to build.
//
// **Where the blank imports live now:** `internal/application/usecases/
// serviceregistrar/imports.go` (package `serviceregistrar`). That
// package depends on `service` (only transitively, via the child
// packages it loads) but `service` does not depend on it — no cycle.
// The composition root blank-imports `serviceregistrar` so the child
// package init() functions run at startup.
//
// **The contract** stays the same: adding a new dynamically-registered
// candidate requires ONE blank import in `serviceregistrar/imports.go`,
// plus the candidate's own `init()` calling [service.Register]. No
// edits to usecases.go (no struct field add), no edits to
// initializers/service.go (no per-candidate construction call).
//
// **Existing typed sub-aggregates** (Audit, Security, Auth) remain
// typed fields on [ServiceUseCases] — they are NOT blank-imported by
// serviceregistrar; they are wired explicitly in
// initializers/service.go.
//
// See docs/plan/20260520-orchestration/decisions.md Q-ORCH-2 and the
// hexagonal-rules.md §8 worked examples (permission_query for
// typed-field; tax_compute for the dynamic registry path that fixed
// the original cycle).
package service
