// Package security hosts composition-level helpers that resolve the
// registered PermissionQuery implementation.
//
// Per docs/plan/20260520-service-domain-migration/ (permission_query
// candidate, 2026-05-20), the previous `consumer.NewPermissionQuery`
// visibility-bridge has been removed from the public consumer surface.
// Its registry-resolution logic moved here so that the composition root
// can wire the service-driven PermissionQuery use case
// (usecases/service/security/get_user_permission_codes.go) without apps
// needing to import `internal/infrastructure/registry`.
package security

import (
	"database/sql"

	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// NewPermissionQueryFromDB returns the registered PermissionQuery backed
// by the provided raw connection, or nil when no RBAC provider has been
// registered (e.g. on non-postgres / non-mock builds). Composition code
// passes the resolved port into the security service use cases
// initializer.
func NewPermissionQueryFromDB(db *sql.DB) securityports.PermissionQuery {
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
