package service

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	auditusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// initServiceAudit wires the service-layer Audit sub-aggregate.
func initServiceAudit(db *sql.DB, authSvc ports.Authorizer, i18nSvc ports.Translator) *auditusecases.UseCases {
	auditSvc := auditServiceFromDB(db)
	return auditusecases.NewUseCases(
		auditusecases.Repositories{AuditService: auditSvc},
		auditusecases.Services{
			Authorizer: authSvc,
			Translator: i18nSvc,
		},
	)
}

// auditServiceFromDB returns the registered AuditService backed by
// the provided raw connection, or nil when no audit provider has been
// registered (e.g. on non-postgres builds). Composition code uses this
// alongside DecorateOperationsWithAudit to materialize the same
// audit service that the decorator will receive.
//
// Keep behaviorally identical with the twin in
// composition/core/container.go; cannot dedupe until core no longer
// imports core/initializers (Codex round-1 §6).
func auditServiceFromDB(db *sql.DB) infraports.AuditService {
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
