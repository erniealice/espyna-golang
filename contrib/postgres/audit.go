//go:build postgresql

package postgres

import (
	"database/sql"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry"
)

func init() {
	// Register the audit service factory — creates the PostgreSQL-backed audit adapter.
	registry.RegisterAuditServiceFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return auditadapter.New(sqlDB)
	})

	// Register the audit-enabled database operations factory — creates a
	// PostgresOperations that automatically logs audit entries on mutations.
	registry.RegisterAuditEnabledOperationsFactory(func(db any, auditSvc any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		svc, ok := auditSvc.(infraports.AuditService)
		if !ok {
			return nil
		}
		return core.NewPostgresOperationsWithAudit(sqlDB, svc)
	})
}
