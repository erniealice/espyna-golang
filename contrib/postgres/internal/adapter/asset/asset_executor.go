//go:build postgresql

package asset

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// PostgresOperations in the core package satisfies this interface via its
// GetExecutor method, which returns interfaces.DBExecutor — the shared
// exported type that avoids the "missing method GetExecutor" panic caused
// by each package previously defining its own unexported dbExecutor copy.
//
// Mirrors entity/entity_executor.go:16. Today the asset adapter routes all
// CRUD/list/page-data calls through r.dbOps.* directly; this interface is
// kept available for future raw-SQL helpers (e.g., a SearchAssetsByName
// like client has) so they can call r.dbOps.(executorProvider).GetExecutor
// without re-defining the type assertion.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface, so that
// future code inside this package can use the short name.
type dbExecutor = interfaces.DBExecutor
