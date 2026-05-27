//go:build mysql

// Package document holds MySQL document adapter implementations.
package document

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
// WorkspaceAwareOperations in the core package satisfies this interface via its
// GetExecutor method, which returns interfaces.DBExecutor.
//
// Mirrors entity/entity_executor.go.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = interfaces.DBExecutor
