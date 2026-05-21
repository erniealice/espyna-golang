package service

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	securityusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/security"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// initServiceSecurity wires the service-layer Security sub-aggregate.
func initServiceSecurity(db *sql.DB, i18nSvc ports.TranslationService) *securityusecases.UseCases {
	permQuery := permissionQueryFromDB(db)
	return securityusecases.NewUseCases(
		securityusecases.Repositories{PermissionQuery: permQuery},
		securityusecases.Services{TranslationService: i18nSvc},
	)
}

// permissionQueryFromDB returns the registered PermissionQuery backed by
// the provided raw connection, or nil when no RBAC provider has been
// registered (e.g. on non-postgres / non-mock builds). Composition code
// passes the resolved port into the security service use cases initializer.
func permissionQueryFromDB(db *sql.DB) securityports.PermissionQuery {
	factory, ok := internalregistry.GetPermissionQueryFactory()
	if !ok || factory == nil {
		return nil
	}
	// The mock RBAC adapter ignores `db` and returns its singleton; the
	// postgres adapter expects a *sql.DB. The factory takes `any` to
	// dodge the cyclic import — see registry/permission_query.go.
	result := factory(db)
	if result == nil {
		return nil
	}
	if pq, ok := result.(securityports.PermissionQuery); ok {
		return pq
	}
	return nil
}
