package registry

import "sync"

// =============================================================================
// PermissionQuery Factory Registry
// =============================================================================
//
// PermissionQueryFactory provides self-registration for the RBAC permission
// query service. Mirrors the AuditServiceFactory pattern: contrib sub-modules
// (e.g. contrib/postgres) register their concrete implementation at init()
// time, and the consumer package discovers it at runtime without build tags.
//
// The factory uses `any` parameters because the concrete type lives in the
// consumer package and cannot be imported here (cyclic).
//
// =============================================================================

var permissionQueryRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterPermissionQueryFactory registers a factory for the permission query
// service. Called from init() in provider-specific packages.
func RegisterPermissionQueryFactory(factory func(db any) any) {
	permissionQueryRegistry.mutex.Lock()
	defer permissionQueryRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterPermissionQueryFactory: factory is nil")
	}
	permissionQueryRegistry.factory = factory
}

// GetPermissionQueryFactory retrieves the registered permission query factory.
// Returns (factory, true) if registered, (nil, false) otherwise.
func GetPermissionQueryFactory() (func(db any) any, bool) {
	permissionQueryRegistry.mutex.RLock()
	defer permissionQueryRegistry.mutex.RUnlock()

	return permissionQueryRegistry.factory, permissionQueryRegistry.factory != nil
}
