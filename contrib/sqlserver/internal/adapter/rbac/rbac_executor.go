//go:build sqlserver

package rbac

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// Mirrors the entity/audit package pattern for consistency.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = sqlexec.DBExecutor
