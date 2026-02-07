package transactions

import (
	"context"

	"leapfor.xyz/espyna/internal/application/ports"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
)

// TransactionServiceAdapter adapts infrastructure TransactionManager to application TransactionService
// This is the bridge between technology-specific infrastructure and technology-agnostic application
type TransactionServiceAdapter struct {
	transactionManager interfaces.TransactionManager
}

// NewTransactionServiceAdapter creates an adapter from TransactionManager to TransactionService
func NewTransactionServiceAdapter(manager interfaces.TransactionManager) ports.TransactionService {
	if manager == nil {
		return ports.NewNoOpTransactionService()
	}

	return &TransactionServiceAdapter{
		transactionManager: manager,
	}
}

// ExecuteInTransaction implements ports.TransactionService
func (a *TransactionServiceAdapter) ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error {
	if a.transactionManager == nil {
		// No transaction manager available - execute without transaction
		return operation(ctx)
	}

	// Convert to infrastructure call with default options
	options := interfaces.DefaultTransactionOptions()
	return a.transactionManager.RunInTransactionWithOptions(ctx, options, operation)
}

// SupportsTransactions implements ports.TransactionService
func (a *TransactionServiceAdapter) SupportsTransactions() bool {
	return a.transactionManager != nil && a.transactionManager.SupportsTransactions()
}

// IsTransactionActive implements ports.TransactionService
func (a *TransactionServiceAdapter) IsTransactionActive(ctx context.Context) bool {
	if a.transactionManager == nil {
		return false
	}

	_, hasTransaction := a.transactionManager.GetTransaction(ctx)
	return hasTransaction
}
