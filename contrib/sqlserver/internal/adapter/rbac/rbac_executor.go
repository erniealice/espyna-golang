//go:build sqlserver

package rbac

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// Mirrors the entity/audit package pattern for consistency.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = interfaces.DBExecutor
