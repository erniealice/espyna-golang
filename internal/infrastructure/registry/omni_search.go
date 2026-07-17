package registry

import "sync"

// =============================================================================
// OmniSearch Factory Registry
// =============================================================================
//
// OmniSearchFactory provides self-registration for the service-driven
// omni-search read capability (service/omni_search). Mirrors the
// OutcomeMatrixFactory pattern: contrib sub-modules (e.g. contrib/postgres)
// register their concrete implementation at init() time, and the composition
// initializer discovers it at runtime without build tags.
//
// The factory uses `any` parameters because the concrete adapter type (which
// implements the GENERATED omni_searchv1.OmniSearchServiceServer interface)
// lives in contrib/postgres and cannot be imported here (cyclic).
//
// =============================================================================

var omniSearchRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterOmniSearchFactory registers a factory for the omni-search query
// service. Called from init() in provider-specific packages.
func RegisterOmniSearchFactory(factory func(db any) any) {
	omniSearchRegistry.mutex.Lock()
	defer omniSearchRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterOmniSearchFactory: factory is nil")
	}
	omniSearchRegistry.factory = factory
}

// GetOmniSearchFactory retrieves the registered omni-search factory. Returns
// (factory, true) if registered, (nil, false) otherwise. On a build where no
// contrib package's init() ran (mock-only / non-postgres), the zero-value
// factory is nil so callers degrade to a nil port (fail-closed).
func GetOmniSearchFactory() (func(db any) any, bool) {
	omniSearchRegistry.mutex.RLock()
	defer omniSearchRegistry.mutex.RUnlock()

	return omniSearchRegistry.factory, omniSearchRegistry.factory != nil
}
