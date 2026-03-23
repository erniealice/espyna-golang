package registry

import "sync"

// =============================================================================
// Audit Service Factory Registry
// =============================================================================
//
// AuditServiceFactory provides self-registration for audit service
// implementations. This allows contrib sub-modules (e.g. contrib/postgres) to
// register their concrete AuditService implementation at init() time,
// and the consumer package can discover it at runtime without build tags.
//
// The factory uses `any` parameters because the concrete type
// (consumer.AuditService) is defined in the consumer package and
// cannot be imported here without creating a circular dependency.
//
// AuditEnabledOperationsFactory provides self-registration for creating
// database operations with audit logging enabled. This allows apps to get
// a DatabaseOperation instance that automatically logs audit entries on
// Create/Update/Delete, by passing back the same raw db + audit service
// that were used to create the audit adapter.
//
// =============================================================================

// auditServiceRegistry holds the registered audit service factory.
var auditServiceRegistry = struct {
	factory func(db any) any
	mutex   sync.RWMutex
}{}

// RegisterAuditServiceFactory registers a factory for creating AuditService.
// This is called from init() in provider-specific packages (e.g. contrib/postgres).
func RegisterAuditServiceFactory(factory func(db any) any) {
	auditServiceRegistry.mutex.Lock()
	defer auditServiceRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterAuditServiceFactory: factory is nil")
	}
	auditServiceRegistry.factory = factory
}

// GetAuditServiceFactory retrieves the registered audit service factory.
// Returns (factory, true) if registered, (nil, false) otherwise.
func GetAuditServiceFactory() (func(db any) any, bool) {
	auditServiceRegistry.mutex.RLock()
	defer auditServiceRegistry.mutex.RUnlock()

	return auditServiceRegistry.factory, auditServiceRegistry.factory != nil
}

// auditEnabledOperationsRegistry holds the registered audit-enabled operations factory.
// factory(db, auditSvc) returns a DatabaseOperation with audit logging enabled.
var auditEnabledOperationsRegistry = struct {
	factory func(db any, auditSvc any) any
	mutex   sync.RWMutex
}{}

// RegisterAuditEnabledOperationsFactory registers a factory for creating
// audit-enabled DatabaseOperation instances.
// This is called from init() in provider-specific packages.
func RegisterAuditEnabledOperationsFactory(factory func(db any, auditSvc any) any) {
	auditEnabledOperationsRegistry.mutex.Lock()
	defer auditEnabledOperationsRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterAuditEnabledOperationsFactory: factory is nil")
	}
	auditEnabledOperationsRegistry.factory = factory
}

// GetAuditEnabledOperationsFactory retrieves the registered audit-enabled operations factory.
// Returns (factory, true) if registered, (nil, false) otherwise.
func GetAuditEnabledOperationsFactory() (func(db any, auditSvc any) any, bool) {
	auditEnabledOperationsRegistry.mutex.RLock()
	defer auditEnabledOperationsRegistry.mutex.RUnlock()

	return auditEnabledOperationsRegistry.factory, auditEnabledOperationsRegistry.factory != nil
}
