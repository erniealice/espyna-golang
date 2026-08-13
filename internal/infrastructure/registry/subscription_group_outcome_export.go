package registry

import "sync"

// =============================================================================
// SubscriptionGroupOutcomeExport Factory Registry
// =============================================================================
//
// SubscriptionGroupOutcomeExportFactory provides self-registration for the
// service-driven subscription-group outcome export read capability
// (service/operation/subscription_group_outcome_export). Mirrors the
// OutcomeMatrixFactory / JobTemplateSummaryFactory pattern: contrib
// sub-modules (e.g. contrib/postgres) register their concrete implementation
// at init() time, and the composition initializer discovers it at runtime
// without build tags.
//
// The factory uses `any` parameters because the concrete adapter type (which
// implements the GENERATED service query port interface) lives in contrib
// providers and cannot be imported here (cyclic).
// =============================================================================

var subscriptionGroupOutcomeExportRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// SubscriptionGroupOutcomeLandingFactoryInput carries the selected provider's
// raw connection plus its canonical entity-to-table/collection mapping. The
// landing registry is deliberately separate from the detailed export registry:
// a provider may supply the non-PII grouped projection without claiming the
// document/matrix export contract.
type SubscriptionGroupOutcomeLandingFactoryInput struct {
	Connection  any
	TableConfig *TableConfig
}

var subscriptionGroupOutcomeLandingRegistry = struct {
	factory func(SubscriptionGroupOutcomeLandingFactoryInput) any
	mutex   sync.RWMutex
}{}

// RegisterSubscriptionGroupOutcomeExportFactory registers a factory for the
// subscription-group outcome export query service. Called from init() in
// provider-specific packages.
func RegisterSubscriptionGroupOutcomeExportFactory(factory func(db any) any) {
	subscriptionGroupOutcomeExportRegistry.mutex.Lock()
	defer subscriptionGroupOutcomeExportRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterSubscriptionGroupOutcomeExportFactory: factory is nil")
	}
	subscriptionGroupOutcomeExportRegistry.factory = factory
}

// GetSubscriptionGroupOutcomeExportFactory retrieves the registered factory.
// Returns (factory, true) if registered, (nil, false) otherwise. On a build
// where no contrib package's init() ran (mock-only / non-postgres), the zero-value
// factory is nil so callers degrade to a nil port (fail-closed).
func GetSubscriptionGroupOutcomeExportFactory() (func(db any) any, bool) {
	subscriptionGroupOutcomeExportRegistry.mutex.RLock()
	defer subscriptionGroupOutcomeExportRegistry.mutex.RUnlock()

	return subscriptionGroupOutcomeExportRegistry.factory, subscriptionGroupOutcomeExportRegistry.factory != nil
}

// RegisterSubscriptionGroupOutcomeLandingFactory registers the selected
// database provider's landing projection factory.
func RegisterSubscriptionGroupOutcomeLandingFactory(factory func(SubscriptionGroupOutcomeLandingFactoryInput) any) {
	subscriptionGroupOutcomeLandingRegistry.mutex.Lock()
	defer subscriptionGroupOutcomeLandingRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterSubscriptionGroupOutcomeLandingFactory: factory is nil")
	}
	subscriptionGroupOutcomeLandingRegistry.factory = factory
}

// GetSubscriptionGroupOutcomeLandingFactory retrieves the independently
// registered landing factory. Absence is a fail-closed nil port.
func GetSubscriptionGroupOutcomeLandingFactory() (func(SubscriptionGroupOutcomeLandingFactoryInput) any, bool) {
	subscriptionGroupOutcomeLandingRegistry.mutex.RLock()
	defer subscriptionGroupOutcomeLandingRegistry.mutex.RUnlock()

	return subscriptionGroupOutcomeLandingRegistry.factory, subscriptionGroupOutcomeLandingRegistry.factory != nil
}
