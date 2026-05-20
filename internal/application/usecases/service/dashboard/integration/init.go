package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
)

// Key is the registry key under which the Integration Dashboard service
// use case sub-aggregate is registered in [service.ServiceUseCases]. The
// dotted form ("dashboard.<X>") is the convention for dynamic-registry
// Wave B dashboard candidates — distinguishes the dashboard namespace
// from top-level service candidates (audit, security, auth, tax).
//
// Internal callers retrieve via [From] or [service.Get]:
//
//	dashUC := integration.From(svc)
//	// equivalent to: service.Get[*integration.UseCases](svc, integration.Key)
//
// **Why registry path, not typed field.** Per Q-ORCH-2-REFINEMENT, the
// rule is callsite-driven: dynamic registry applies when ALL callers
// live inside the espyna package tree. Codex P1 verified
// `rg -n "GetIntegrationDashboardPageData" apps/service-admin --type go`
// returns 0. This makes P1.C.10 the canonical Wave B exemplar of the
// registry path (per wave-b-surface-map.md).
//
// **Future hybra rewire requires an espyna-internal adapter shim, NOT
// a direct `integration.From()` call** (codex review P1 2026-05-20).
// The hybra integration view at
// `packages/hybra-golang/views/integration/module.go:27` CANNOT import
// `github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/integration`
// because Go's `internal/` visibility rule restricts that path to
// importers under `espyna-golang/...`. When hybra is rewired in a
// follow-up commit, the wiring MUST go through an espyna-internal
// adapter (e.g., a public `composition/dashboard/integration/wrapper.go`
// that exposes a hybra-facing function signature using non-internal
// types) — same shape as `composition/auth/wrapper.go` and
// `composition/security/permission_query.go`.
//
// **Why no SetEntityCompute global setter.** Unlike tax_compute (the
// other registry-path exemplar at usecases/service/tax/init.go), the
// Integration Dashboard's entity-layer use case is constructed in-place
// by the factory — no captured typed dep from an external aggregate is
// needed. This matches the cleaner permission_query reference impl
// shape; the package-level setter pattern is intentionally avoided per
// the hexagonal-rules.md §8 caveat ("future authors should NOT blindly
// copy SetEntityCompute"). When future Wave B work wires real provider
// stats hooks at the entity layer, the factory below extends to thread
// them via either [service.Deps] (if the dep generalizes) or a
// candidate-specific closure built in the composition root.
const Key = "dashboard.integration"

func init() {
	service.Register(Key, func(deps *service.Deps) any {
		return NewUseCases()
	})
}

// From is the typed accessor companion to the registry registration.
// Returns nil if the sub-aggregate is unregistered (e.g. the
// serviceregistrar blank import is not loaded, which would be a wiring
// bug in the composition root).
func From(s *service.ServiceUseCases) *UseCases {
	return service.Get[*UseCases](s, Key)
}
