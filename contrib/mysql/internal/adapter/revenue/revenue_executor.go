//go:build mysql

package revenue

import (
	"context"
	"database/sql"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
)

// executorProvider is implemented by WorkspaceAwareOperations; it lets raw-SQL
// methods inside this package obtain a DBExecutor that participates in the
// current transaction when one is active.
type executorProvider interface {
	GetExecutor(ctx context.Context) mysqlCore.DBExecutor
}

// getDB extracts the underlying *sql.DB from a DatabaseOperation if the
// implementation exposes GetDB(). Falls back to nil; callers must check.
func getDB(dbOps interfaces.DatabaseOperation) *sql.DB {
	if provider, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		return provider.GetDB()
	}
	return nil
}
