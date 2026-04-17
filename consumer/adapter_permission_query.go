package consumer

import (
	"context"
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// PermissionQuery is the consumer-visible handle for RBAC permission lookups.
// Matches internal/application/ports/security.PermissionQuery but is re-exported
// here so consumer apps don't have to import internal/.
type PermissionQuery interface {
	GetUserPermissionCodes(ctx context.Context, userID, workspaceID string) ([]string, error)
}

// NewPermissionQuery returns the registered permission query implementation.
// Returns nil if no provider has registered a factory (e.g. no postgresql
// build tag). Callers should handle nil gracefully.
//
// Currently PostgreSQL-only; Firestore / Mock impls will register as they're
// added in Phase 3 of the auth/database refactor.
func NewPermissionQuery(db *sql.DB) PermissionQuery {
	factory, ok := internalregistry.GetPermissionQueryFactory()
	if !ok {
		return nil
	}
	result := factory(db)
	if pq, ok := result.(security.PermissionQuery); ok {
		return pq
	}
	return nil
}
