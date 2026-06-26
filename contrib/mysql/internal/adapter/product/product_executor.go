//go:build mysql

// Package product holds MySQL product domain adapter implementations.
package product

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// WorkspaceAwareOperations in the core package satisfies this interface via its
// GetExecutor method, which returns sqlexec.DBExecutor.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface, so that
// existing code inside this package can continue to use the short name.
type dbExecutor = sqlexec.DBExecutor
