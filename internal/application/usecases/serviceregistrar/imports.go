// Package serviceregistrar is the central blank-import host for
// dynamically-registered service-driven candidate packages.
//
// **Why this package exists:** the registry (Register/Get/Deps) lives
// in `internal/application/usecases/service` because the [ServiceUseCases]
// aggregate consumes the registered components. Each candidate package
// (e.g. `service/tax`) imports `package service` to call
// [service.Register] in its `init()`. If a blank-import file in
// `package service` then imported those children, Go would detect the
// cycle and refuse to build.
//
// This package sits OUTSIDE `service` so that:
//
//   - `serviceregistrar` imports child packages (transitive: `service`).
//   - children import `service` to register.
//   - `service` imports nothing from `serviceregistrar` (no cycle).
//
// **The contract for adding a new candidate:** add ONE blank import
// here, plus the candidate's `init()` calling [service.Register]. No
// edits to `service/usecases.go` (no struct field add), no edits to
// `initializers/service.go` (no per-candidate construction call).
//
// **Loading:** the composition root blank-imports this package
// (`internal/composition/core/usecases.go`) so that every candidate's
// `init()` runs at startup before [service.NewServiceUseCases] is
// called.
//
// See docs/wiki/articles/hexagonal-rules.md §8 (tax_compute worked
// example) for the canonical candidate-package shape.
package serviceregistrar

import (
	// Wave 3 / Plan 2 candidate — tax_compute. Espyna-internal caller
	// (recognize_revenue_from_subscription.go), so the dynamic registry
	// path applies per Q-ORCH-2-REFINEMENT. The package's init() calls
	// service.Register("tax", factory); the composition root calls
	// servicetax.SetEntityCompute(...) before NewServiceUseCases so the
	// factory has the entity-layer compute use case to wrap.
	_ "github.com/erniealice/espyna-golang/internal/application/usecases/service/tax"

	// Wave B P1.C.10 candidate — Integration Dashboard. Zero
	// apps/service-admin callsites (verified by codex P1); only
	// hybra-golang/views/integration consumes it. Espyna-package-internal,
	// so the dynamic-registry path applies per Q-ORCH-2-REFINEMENT. The
	// package's init() calls service.Register("dashboard.integration",
	// factory). No SetEntityCompute analog — the factory constructs the
	// entity-layer use case in-place (cleaner permission_query-style
	// pattern, NOT tax_compute's global-state pattern; see
	// service/dashboard/integration/init.go doc-comment for rationale).
	_ "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/integration"

	// Future dynamic-registry candidates (dashboards, reporting) add
	// their blank import here as they ship. Typed-field candidates
	// (Audit, Security, Auth) do NOT live here — they are wired
	// explicitly in initializers/service.go.
)
