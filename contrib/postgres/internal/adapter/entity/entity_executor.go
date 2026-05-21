//go:build postgresql

package entity

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// PostgresOperations in the core package satisfies this interface via its
// GetExecutor method, which returns interfaces.DBExecutor — the shared
// exported type that avoids the "missing method GetExecutor" panic caused
// by each package previously defining its own unexported dbExecutor copy.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface, so that
// existing code inside this package can continue to use the short name.
type dbExecutor = interfaces.DBExecutor
