package registry

import "sync"

// =============================================================================
// Assignee Query Factory Registry
// =============================================================================
//
// Engine Identity Bridge (Q-EIB-BRIDGE): provides self-registration for the
// AssigneeQueryRepository implementation. The postgres adapter registers its
// concrete PostgresAssigneeQueryRepository at init() time via
// RegisterAssigneeQueryFactory, and the container resolves it at runtime
// without importing the build-tagged adapter.
//
// The factory uses `any` parameters because the concrete adapter lives behind
// a build tag (//go:build postgresql) and cannot be imported by the
// dialect-neutral container.
//
// =============================================================================

// assigneeQueryRegistry holds the registered assignee query factory.
var assigneeQueryRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterAssigneeQueryFactory registers a factory for creating an
// AssigneeQueryRepository from a database connection.
// Called from init() in contrib/postgres/internal/adapter/workflow/activity_assignee.go.
func RegisterAssigneeQueryFactory(factory func(db any) any) {
	assigneeQueryRegistry.mutex.Lock()
	defer assigneeQueryRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterAssigneeQueryFactory: factory is nil")
	}
	assigneeQueryRegistry.factory = factory
}

// GetAssigneeQueryFactory retrieves the registered assignee query factory.
// Returns (factory, true) if registered, (nil, false) otherwise.
func GetAssigneeQueryFactory() (func(db any) any, bool) {
	assigneeQueryRegistry.mutex.RLock()
	defer assigneeQueryRegistry.mutex.RUnlock()

	return assigneeQueryRegistry.factory, assigneeQueryRegistry.factory != nil
}
