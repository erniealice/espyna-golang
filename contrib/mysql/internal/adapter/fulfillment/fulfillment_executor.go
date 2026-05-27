//go:build mysql

// Package fulfillment holds MySQL fulfillment domain adapter implementations.
package fulfillment

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
