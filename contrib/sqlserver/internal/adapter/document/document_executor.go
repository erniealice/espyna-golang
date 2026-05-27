//go:build sqlserver

package document

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// SQLServerOperations in the core package satisfies this interface via its
// GetExecutor method, which returns interfaces.DBExecutor.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = interfaces.DBExecutor
