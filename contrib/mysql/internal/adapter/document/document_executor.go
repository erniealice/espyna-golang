//go:build mysql

// Package document holds MySQL document adapter implementations.
package document

import (
	"context"

	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// executorProvider provides a transaction-aware database executor.
// WorkspaceAwareOperations in the core package satisfies this interface via its
// GetExecutor method, which returns sqlexec.DBExecutor.
//
// Mirrors entity/entity_executor.go.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = sqlexec.DBExecutor
