// Package transactions re-exports internal transaction adapter types for use by contrib sub-modules.
package transactions

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/transactions"
)

type TransactionServiceAdapter = internal.TransactionServiceAdapter

var NewTransactionServiceAdapter = internal.NewTransactionServiceAdapter
