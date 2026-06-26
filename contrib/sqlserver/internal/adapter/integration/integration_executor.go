//go:build sqlserver

package integration

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// SQLServerOperations in the core package satisfies this interface via its
// GetExecutor method, which returns sqlexec.DBExecutor.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = sqlexec.DBExecutor
