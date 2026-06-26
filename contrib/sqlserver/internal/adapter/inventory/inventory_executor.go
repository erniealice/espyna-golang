//go:build sqlserver

package inventory

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = sqlexec.DBExecutor
