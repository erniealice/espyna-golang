package registry

import "sync"

// =============================================================================
// JobListTabSupport Factory Registry
// =============================================================================
//
// JobListTabSupportFactory provides self-registration for the job-list tabstrip
// support read (the 20260718 courses-list-perf Rank-1 counter-plan: ONE UNION-ALL
// statement replacing the 12 generic-List category/template statements). Mirrors
// the JobTemplateSummaryFactory / AssigneeQueryFactory pattern: contrib sub-
// modules (e.g. contrib/postgres) register their concrete implementation at
// init() time, and the composition initializer discovers it at runtime without
// build tags.
//
// The factory uses `any` parameters because the concrete adapter lives behind a
// build tag (//go:build postgresql) and cannot be imported by the dialect-neutral
// container.
//
// =============================================================================

var jobListTabSupportRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterJobListTabSupportFactory registers a factory for the job-list
// tab-support query service. Called from init() in provider-specific packages.
func RegisterJobListTabSupportFactory(factory func(db any) any) {
	jobListTabSupportRegistry.mutex.Lock()
	defer jobListTabSupportRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterJobListTabSupportFactory: factory is nil")
	}
	jobListTabSupportRegistry.factory = factory
}

// GetJobListTabSupportFactory retrieves the registered job-list tab-support
// factory. Returns (factory, true) if registered, (nil, false) otherwise. On a
// build where no contrib package's init() ran (mock-only / non-postgres), the
// zero-value factory is nil so callers degrade to a nil port (fail-closed).
func GetJobListTabSupportFactory() (func(db any) any, bool) {
	jobListTabSupportRegistry.mutex.RLock()
	defer jobListTabSupportRegistry.mutex.RUnlock()

	return jobListTabSupportRegistry.factory, jobListTabSupportRegistry.factory != nil
}
