//go:build mysql

// Package inventory holds MySQL inventory domain adapter implementations.
package inventory

import (
	"context"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// executorProvider provides a transaction-aware database executor.
type executorProvider interface {
	GetExecutor(ctx context.Context) interfaces.DBExecutor
}

// dbExecutor is a package-local alias for the shared interface.
type dbExecutor = interfaces.DBExecutor
