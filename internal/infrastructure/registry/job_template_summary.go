package registry

import "sync"

// =============================================================================
// JobTemplateSummary Factory Registry
// =============================================================================
//
// JobTemplateSummaryFactory provides self-registration for the service-driven
// job-template-summary read capability (service/operation/job_template_summary).
// Mirrors the OutcomeMatrixFactory / PermissionQueryFactory pattern: contrib
// sub-modules (e.g. contrib/postgres) register their concrete implementation at
// init() time, and the composition initializer discovers it at runtime without
// build tags.
//
// The factory uses `any` parameters because the concrete adapter type (which
// implements the GENERATED operationv1.JobTemplateSummaryServiceServer
// interface) lives in contrib/postgres and cannot be imported here (cyclic).
//
// =============================================================================

var jobTemplateSummaryRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterJobTemplateSummaryFactory registers a factory for the
// job-template-summary query service. Called from init() in provider-specific
// packages.
func RegisterJobTemplateSummaryFactory(factory func(db any) any) {
	jobTemplateSummaryRegistry.mutex.Lock()
	defer jobTemplateSummaryRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterJobTemplateSummaryFactory: factory is nil")
	}
	jobTemplateSummaryRegistry.factory = factory
}

// GetJobTemplateSummaryFactory retrieves the registered job-template-summary
// factory. Returns (factory, true) if registered, (nil, false) otherwise. On a
// build where no contrib package's init() ran (mock-only / non-postgres), the
// zero-value factory is nil so callers degrade to a nil port (fail-closed).
func GetJobTemplateSummaryFactory() (func(db any) any, bool) {
	jobTemplateSummaryRegistry.mutex.RLock()
	defer jobTemplateSummaryRegistry.mutex.RUnlock()

	return jobTemplateSummaryRegistry.factory, jobTemplateSummaryRegistry.factory != nil
}
