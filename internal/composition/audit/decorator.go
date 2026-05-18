// Package audit hosts composition-level helpers that decorate raw
// database operations with audit logging.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/phases.md §P1.D
// (D-iv), the previous `consumer.NewDatabaseAdapterWithAudit` factory
// has been removed from the public consumer surface. Its logic moved
// here so that the composition root can transparently wrap
// `GetDatabaseOperations()` with audit-decorated ops — apps never need
// to know the decorator exists.
package audit

import (
	"database/sql"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// DecorateOperationsWithAudit returns a DatabaseOperation impl wrapped
// with audit logging when both an audit service and an audit-enabled
// operations factory have been registered (e.g. via contrib/postgres
// init()).
//
// Returns the original `ops` value unchanged when:
//   - the audit-enabled operations factory is unregistered, OR
//   - auditSvc is nil, OR
//   - the audit-decorated factory returns nil (mis-typed inputs).
//
// The `db` value is forwarded to the factory as-is — callers are
// expected to pass the same raw connection that backs `ops`.
func DecorateOperationsWithAudit(ops any, db *sql.DB, auditSvc infraports.AuditService) any {
	if ops == nil || db == nil || auditSvc == nil {
		return ops
	}
	factory, ok := internalregistry.GetAuditEnabledOperationsFactory()
	if !ok || factory == nil {
		return ops
	}
	decorated := factory(db, auditSvc)
	if decorated == nil {
		return ops
	}
	return decorated
}

// NewAuditServiceFromDB returns the registered AuditService backed by
// the provided raw connection, or nil when no audit provider has been
// registered (e.g. on non-postgres builds). Composition code uses this
// alongside DecorateOperationsWithAudit to materialize the same
// audit service that the decorator will receive.
func NewAuditServiceFromDB(db *sql.DB) infraports.AuditService {
	if db == nil {
		return nil
	}
	factory, ok := internalregistry.GetAuditServiceFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if svc, ok := result.(infraports.AuditService); ok {
		return svc
	}
	return nil
}
