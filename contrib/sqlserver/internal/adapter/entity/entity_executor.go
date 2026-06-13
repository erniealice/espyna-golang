//go:build sqlserver

package entity

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// SQLServerOperations in the core package satisfies this interface via its
// GetExecutor method, which returns sqlexec.DBExecutor — the shared
// exported type that avoids the "missing method GetExecutor" panic caused
// by each package previously defining its own unexported dbExecutor copy.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface, so that
// existing code inside this package can continue to use the short name.
type dbExecutor = sqlexec.DBExecutor
