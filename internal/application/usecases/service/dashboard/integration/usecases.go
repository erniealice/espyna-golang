// Package integration hosts the service-driven Integration Dashboard use
// case.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// service-driven domain category) and the Wave B candidate map at
// docs/plan/20260520-service-domain-migration/wave-b-surface-map.md
// §P1.C.10, Integration Dashboard is a cross-provider read projection: it
// aggregates counts / trends / recent errors across the four integration
// providers (payment, email, scheduler, tabular) and owns no aggregate of
// its own. That makes it a service-driven domain.
//
// Its proto contract lives at `proto/v1/service/dashboard/integration/
// dashboard.proto` (Q-SDM-DASHBOARD-LAYOUT subdir-per-dashboard +
// Q-SDM-DASHBOARD-SHARED-TYPES per-dashboard messages).
//
// **Registry-path exemplar.** P1.C.10 is the canonical Wave B exemplar for
// the dynamic-registry path (Q-ORCH-2 / Q-ORCH-2-REFINEMENT). Codex P1
// verified zero `apps/service-admin` callsites for the underlying
// `GetIntegrationDashboardPageData` use case; only the `hybra-golang`
// integration view consumes it. Apps cannot name `*UseCases` (it lives
// under `internal/`), but hybra is a package — the Go `internal/` rule
// does not block hybra from naming the registry's generic type parameter
// when the view-side migration happens.
//
// This package follows the cleaner permission_query reference impl
// structure rather than tax_compute's `SetEntityCompute` global-state
// pattern. The entity-layer `GetIntegrationDashboardPageDataUseCase` is
// nil-safe by construction (returns an empty response when its stats
// queries port is unwired), so the factory can build a fresh entity-layer
// use case at registry-resolve time with the deps it already has — no
// package-level mutable bootstrap state required.
//
// **Bootstrap scope (Wave B pilot).** The first registry-resolved
// integration dashboard intentionally constructs the entity-layer use case
// with a nil `IntegrationStatsQueries` port — matching the current pattern
// at usecases/integration/usecases.go:107 where the existing
// `IntegrationUseCases.Dashboard` is also constructed with nil. The
// service-layer wrapper therefore renders empty-state proto responses by
// design, and the view degrades gracefully (stats=0, no providers,
// flat-zero trend, no errors). When future work wires real provider stats
// hooks at the entity layer, the factory below extends to thread them
// through to the entity-layer use case — no service-layer contract change
// needed.
package integration

// UseCases aggregates every service-driven Integration Dashboard use case.
type UseCases struct {
	GetIntegrationDashboard *GetIntegrationDashboardUseCase
}

// NewUseCases wires every Integration Dashboard service use case.
//
// The entity-layer `GetIntegrationDashboardPageDataUseCase` is built
// internally inside the factory (see init.go). Construction is currently
// dep-free at the entity layer because the queries port stays nil until
// provider stats hooks land — see package doc.
func NewUseCases() *UseCases {
	return &UseCases{
		GetIntegrationDashboard: NewGetIntegrationDashboardUseCase(),
	}
}
