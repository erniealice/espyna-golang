// Package service hosts the service-driven domain use case sub-aggregates.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// LOCKED), service-driven domains are first-class but distinct from the
// entity-driven 14-layer canon: they have proto contracts and use cases
// but no entityid/provider/route. This package is the Layer-7 anchor for
// that category. Phase 1 anchors it with one sub-aggregate (audit); the
// follow-up plan migrates reporting/auth/security here too.
//
// **Typed field pattern** — All service-driven candidates are wired as
// typed fields on ServiceUseCases. This pattern is required for stability
// and maintainability. Each candidate package New function is called
// directly in the composition root (initializers/service/service.go) and
// passed as an argument to NewServiceUseCases. See the per-candidate
// initializer functions (initServiceAudit, initServiceSecurity, etc.) for
// the wiring pattern.
//
// **Wave B umbrella sub-aggregates (Dashboard, Reporting):** the dashboard
// and reporting candidates expand the typed-field pattern with a small
// twist — instead of adding 11 + 5 separate fields on this aggregate
// (high parallel-merge contention), the Dashboard and Reporting
// umbrellas each hold a single field here that points to a sub-
// aggregate package (`service/dashboard` and `service/reporting`)
// whose own struct hosts the per-candidate typed pointers. Each Wave B
// candidate edits ONLY its umbrella's small `usecases.go` (low
// contention, near-zero parallel conflict) + ONE line in
// `initializers/service.go`. The Audit/Security/Auth fields keep the
// direct typed-field shape (their packages were locked before the
// umbrella pattern existed).
package service

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/amortization"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/performance"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/security"
	servicetax "github.com/erniealice/espyna-golang/internal/application/usecases/service/tax"
)

// ServiceUseCases aggregates every service-driven use case package.
type ServiceUseCases struct {
	Audit     *audit.UseCases              // Phase 1.D (20260518)
	Security  *security.UseCases           // Phase 1.A (20260520, permission_query)
	Auth      *serviceauth.UseCases        // Wave 3 / Plan 2 (20260520, auth-session)
	Dashboard *dashboard.DashboardUseCases // Wave B P1.C (20260520, 11 dashboard candidates)
	Reporting *reporting.ReportingUseCases // Wave B P1.E (20260520, 5 ledger reporting groups)

	// Performance Evaluation (20260604 v1) service-layer orchestration.
	Performance *performance.UseCase // servicing-gated panel cross-join

	// Tax compute service (20260520 Plan 2 / Q-SDM-TAX) — wraps the
	// entity-layer use case with a proto contract. Nil-safe: when unset,
	// the tax-compute hook in recognize-revenue degrades gracefully.
	Tax *servicetax.UseCases

	// Amortization schedule service (20260604 Wave B, pure computation) —
	// wrapped from the shared package. Nil-safe: when unset, amortization
	// computations degrade to nil.
	Amortization *amortization.UseCases
}

// NewServiceUseCases wires every service-driven sub-aggregate. All typed
// fields (Audit, Security, Auth, Dashboard, Reporting, Tax, Amortization)
// are passed explicitly.
//
// Sub-aggregates may be nil when the relevant infrastructure provider is
// unregistered.
//
// Dashboard and Reporting are umbrella aggregates — each holds per-
// candidate typed pointer fields that Wave B candidate agents populate
// independently. See package docs on `service/dashboard` and
// `service/reporting` for the per-candidate edit pattern.
func NewServiceUseCases(
	audit *audit.UseCases,
	security *security.UseCases,
	auth *serviceauth.UseCases,
	dash *dashboard.DashboardUseCases,
	rep *reporting.ReportingUseCases,
	perf *performance.UseCase,
	tax *servicetax.UseCases,
	amort *amortization.UseCases,
) *ServiceUseCases {
	return &ServiceUseCases{
		Audit:        audit,
		Security:     security,
		Auth:         auth,
		Dashboard:    dash,
		Reporting:    rep,
		Performance:  perf,
		Tax:          tax,
		Amortization: amort,
	}
}
