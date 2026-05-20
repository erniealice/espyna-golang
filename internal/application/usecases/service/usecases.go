// Package service hosts the service-driven domain use case sub-aggregates.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// LOCKED), service-driven domains are first-class but distinct from the
// entity-driven 14-layer canon: they have proto contracts and use cases
// but no entityid/provider/route. This package is the Layer-7 anchor for
// that category. Phase 1 anchors it with one sub-aggregate (audit); the
// follow-up plan migrates reporting/auth/security here too.
//
// **Two coexisting registration patterns** (see docs/plan/20260520-orchestration/
// decisions.md Q-ORCH-2 and Q-ORCH-2-REFINEMENT):
//
//  1. Typed fields (Audit, Security, Auth) — used when the sub-aggregate is
//     **called from `apps/*`**. Required pattern for app-visible candidates
//     because Go's `internal/` visibility rule prevents apps from naming
//     `*<X>.UseCases` as the generic type parameter `T` in
//     `service.Get[T](svc, "<key>")`. Stable, type-safe; the cost is one
//     struct field add + one [NewServiceUseCases] argument per candidate.
//     The audit / security / permission_query / auth migrations are the
//     canonical exemplars.
//
//  2. Dynamic components (via [Register] / [Get]) — used **only when ALL
//     callsites live inside `packages/espyna-golang/`** (espyna-internal
//     callers). Each candidate's package init() calls
//     service.Register("<key>", factory); blank imports are owned by the
//     sibling [serviceregistrar] package (NOT this package's
//     imports.go, which is a docstring-only file kept to explain the
//     cycle-break). Adding a new dynamic candidate requires ONE blank
//     import in [serviceregistrar/imports.go] — no struct edits.
//     The tax_compute candidate is the canonical exemplar (caller:
//     RecognizeRevenueFromSubscription, espyna-internal).
//
// **Rule of thumb (Q-ORCH-2-REFINEMENT):** check the candidate's callsites.
// Any caller under `apps/*`? Use pattern (1) typed field. All callers under
// `packages/espyna-golang/`? Pattern (2) dynamic registry is available; both
// patterns are acceptable but pattern (2) avoids shared-file conflicts when
// many candidates ship in parallel. New candidates should NOT default to
// pattern (2) — the rule is callsite-driven, not preference-driven.
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
	"database/sql"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/security"
)

// Deps holds the typed dependencies that dynamically-registered candidates
// may need at construction time. Extend this struct only when a candidate
// genuinely needs a new typed dep — avoid making it a junk drawer.
type Deps struct {
	DB                   *sql.DB
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ServiceUseCases aggregates every service-driven use case package.
type ServiceUseCases struct {
	Audit     *audit.UseCases                // Phase 1.D (20260518)
	Security  *security.UseCases             // Phase 1.A (20260520, permission_query)
	Auth      *serviceauth.UseCases          // Wave 3 / Plan 2 (20260520, auth-session)
	Dashboard *dashboard.DashboardUseCases   // Wave B P1.C (20260520, 11 dashboard candidates)
	Reporting *reporting.ReportingUseCases   // Wave B P1.E (20260520, 5 ledger reporting groups)

	components map[string]any // dynamic registry — see [Register] / [Get]
}

var (
	factoriesMu sync.Mutex
	factories   = map[string]func(*Deps) any{}
)

// Register adds a factory for a dynamically-registered service-driven
// candidate. Called from a candidate package's init() function.
//
// The factory receives the typed [Deps] and returns the candidate's
// sub-aggregate value (usually *<package>.UseCases). The value is stored
// in [ServiceUseCases]'s components map under the given key.
//
// Per the canonical pattern (docs/wiki/articles/hexagonal-rules.md §8
// Worked examples), each candidate package owns its key constant +
// accessor function:
//
//	package auth
//	const Key = "auth"
//	func init() {
//	    service.Register(Key, func(deps *service.Deps) any {
//	        return New(deps.AuthorizationService, deps.TranslationService)
//	    })
//	}
//	func From(s *service.ServiceUseCases) *UseCases {
//	    return service.Get[*UseCases](s, Key)
//	}
//
// Then [imports.go] adds:
//
//	import _ "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
func Register(key string, factory func(*Deps) any) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[key] = factory
}

// Get returns the registered component under the given key, type-asserted
// to T. Returns the zero value of T if no component is registered or the
// type assertion fails.
func Get[T any](s *ServiceUseCases, key string) T {
	var zero T
	if s == nil || s.components == nil {
		return zero
	}
	if v, ok := s.components[key].(T); ok {
		return v
	}
	return zero
}

// NewServiceUseCases wires every service-driven sub-aggregate. The typed
// fields (Audit, Security, Auth, Dashboard, Reporting) are passed
// explicitly; dynamically-registered candidates are constructed from
// the registered factories using deps.
//
// Sub-aggregates may be nil when the relevant infrastructure provider is
// unregistered. deps may be nil if no dynamically-registered candidate
// requires construction (e.g., in unit tests that only touch typed fields).
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
	deps *Deps,
) *ServiceUseCases {
	uc := &ServiceUseCases{
		Audit:      audit,
		Security:   security,
		Auth:       auth,
		Dashboard:  dash,
		Reporting:  rep,
		components: make(map[string]any),
	}

	if deps == nil {
		return uc
	}

	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	for key, factory := range factories {
		uc.components[key] = factory(deps)
	}

	return uc
}
