//go:build sqlserver

package tax

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// WorkspaceAwareOperations in the core package satisfies this interface via its
// GetExecutor method.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = sqlexec.DBExecutor
