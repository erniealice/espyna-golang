package tax

import (
	"sync"

	taxcompute "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
)

// Key is the registry key under which tax service-driven use cases are
// registered in [service.ServiceUseCases]. Internal callers (espyna only,
// per Q-ORCH-2-REFINEMENT — Go internal/ rule prevents apps from naming
// *tax.UseCases) retrieve via [From] or [service.Get]:
//
//	taxUC := tax.From(svc.Service)
//	// equivalent to: service.Get[*tax.UseCases](svc.Service, tax.Key)
//
// Per Q-SDM-TAX (LOCKED 2026-05-20), tax_compute is the worked example
// for espyna-internal candidates using the dynamic registry pattern.
//
// **INTENTIONAL PARTIAL SCOPE.** Per Q-SDM-TAX, only one of the three
// known callers of the entity-layer ComputeTaxesForRevenue use case is
// rewired through this service-driven wrapper. The other two are
// intentionally left wired to the entity-layer use case for now:
//
//  1. Rewired (this migration):
//     packages/espyna-golang/internal/application/usecases/domain/revenue/revenue/
//     recognize_revenue_from_subscription.go — the
//     RecognizeRevenueFromSubscription post-persist hook now invokes
//     servicetax.From(serviceUC).ComputeTaxesForRevenue.ExecuteForRevenue
//     via its narrow ComputeTaxesForRevenueInvoker interface. Wiring lives
//     in internal/composition/core/usecases.go (search "service-driven
//     path"). This is the Q7 §8 worked example caller — the only one
//     Q-SDM-TAX explicitly requires migrating.
//
//  2. Deferred — RecomputeTaxes (admin direct entity-layer caller).
//     packages/espyna-golang/internal/application/usecases/domain/revenue/revenue/
//     recompute_taxes.go — invoked from admin recompute flows. Stays
//     wired to the entity-layer ComputeTaxesForRevenue use case via
//     SetComputeTaxes in internal/composition/core/usecases.go (the
//     "Tax compute wired into revenue domain (… → RecomputeTaxes)" log
//     line). Migration deferred because the caller is a Layer-7 admin
//     use case, not a service-shape consumer; rewiring it provides no
//     new contract value while increasing surface area.
//
//  3. Deferred — CreateRevenue post-persist hook (dormant).
//     packages/espyna-golang/internal/application/usecases/domain/revenue/revenue/
//     create_revenue.go — the SetComputeTaxes hook on CreateRevenue
//     is currently dormant (no production flow invokes it; only present
//     for Phase D wiring symmetry with the recognize path). Stays
//     entity-direct until the hook becomes load-bearing; at that point
//     rewire alongside the recognize-side caller.
//
// Bootstrap shape — tax is the FIRST registry user, and its construction
// requires a typed dependency (the entity-layer ComputeTaxesForRevenue
// use case) that the canonical [service.Deps] struct does not carry. To
// avoid editing the shared [service.Deps] for every future candidate's
// specific deps, this package exposes a package-level [SetEntityCompute]
// setter the composition root calls BEFORE [service.NewServiceUseCases]
// runs. The registered factory reads the captured value at construction
// time. When the entity-layer compute is nil (no SQL provider), the
// factory returns nil and the registry lookup degrades gracefully.
//
// **Caveat — package-level mutable bootstrap state.** [SetEntityCompute]
// is concurrency-safe (RWMutex-guarded) but it IS global state. Future
// authors should NOT blindly copy this pattern for new dynamic-registry
// candidates without weighing alternatives. The cleaner long-term shape
// would be either: (a) carry the typed entity dep on [service.Deps] when
// it generalizes across candidates (rare; risks junk-drawer drift), or
// (b) capture the dep inside a candidate-specific factory closure built
// in the composition root (requires the candidate to expose a factory
// constructor instead of relying on init() registration). Tax adopted
// the package-level setter only because it is the FIRST registry user
// and the canonical bootstrap shape was not yet settled. Revisit if a
// second candidate develops the same need.
const Key = "tax"

var (
	entityComputeMu sync.RWMutex
	entityCompute   *taxcompute.ComputeTaxesForRevenueUseCase
)

// SetEntityCompute captures the entity-layer ComputeTaxesForRevenue use
// case for use by the registered factory. The composition root calls
// this AFTER initializing the entity-layer tax aggregate and BEFORE
// building the service use case aggregate. Passing nil clears the value
// (and the factory will then return nil, leaving the sub-aggregate
// unwired).
func SetEntityCompute(uc *taxcompute.ComputeTaxesForRevenueUseCase) {
	entityComputeMu.Lock()
	defer entityComputeMu.Unlock()
	entityCompute = uc
}

func init() {
	service.Register(Key, func(deps *service.Deps) any {
		entityComputeMu.RLock()
		ec := entityCompute
		entityComputeMu.RUnlock()
		if ec == nil {
			return (*UseCases)(nil)
		}
		return NewUseCases(Repositories{EntityCompute: ec})
	})
}

// From is the typed accessor companion to the registry registration.
// Returns nil if the tax sub-aggregate is unregistered or the
// entity-layer compute was never captured.
func From(s *service.ServiceUseCases) *UseCases {
	return service.Get[*UseCases](s, Key)
}
