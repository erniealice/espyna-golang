//go:build sqlserver

package tenancy

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// Mirrors the entity package pattern: sqlserverCore.WorkspaceAwareOperations
// satisfies this interface via its GetExecutor method.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = interfaces.DBExecutor
