package registry

import "sync"

// =============================================================================
// OutcomeMatrix Factory Registry
// =============================================================================
//
// OutcomeMatrixFactory provides self-registration for the service-driven
// outcome-matrix read capability (service/operation/outcome_matrix). Mirrors
// the PermissionQueryFactory pattern: contrib sub-modules (e.g.
// contrib/postgres) register their concrete implementation at init() time, and
// the composition initializer discovers it at runtime without build tags.
//
// The factory uses `any` parameters because the concrete adapter type (which
// implements the GENERATED operationv1.OutcomeMatrixServiceServer interface)
// lives in contrib/postgres and cannot be imported here (cyclic).
//
// =============================================================================

var outcomeMatrixRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterOutcomeMatrixFactory registers a factory for the outcome-matrix query
// service. Called from init() in provider-specific packages.
func RegisterOutcomeMatrixFactory(factory func(db any) any) {
	outcomeMatrixRegistry.mutex.Lock()
	defer outcomeMatrixRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterOutcomeMatrixFactory: factory is nil")
	}
	outcomeMatrixRegistry.factory = factory
}

// GetOutcomeMatrixFactory retrieves the registered outcome-matrix factory.
// Returns (factory, true) if registered, (nil, false) otherwise. On a build
// where no contrib package's init() ran (mock-only / non-postgres), the
// zero-value factory is nil so callers degrade to a nil port (fail-closed).
func GetOutcomeMatrixFactory() (func(db any) any, bool) {
	outcomeMatrixRegistry.mutex.RLock()
	defer outcomeMatrixRegistry.mutex.RUnlock()

	return outcomeMatrixRegistry.factory, outcomeMatrixRegistry.factory != nil
}
