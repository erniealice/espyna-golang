package transactions

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
)

// TransactionServiceAdapter adapts infrastructure TransactionManager to application Transactor
// This is the bridge between technology-specific infrastructure and technology-agnostic application
type TransactionServiceAdapter struct {
	transactionManager interfaces.TransactionManager
}

// NewTransactionServiceAdapter creates an adapter from TransactionManager to Transactor
func NewTransactionServiceAdapter(manager interfaces.TransactionManager) ports.Transactor {
	if manager == nil {
		return ports.NewNoOpTransactor()
	}

	return &TransactionServiceAdapter{
		transactionManager: manager,
	}
}

// ExecuteInTransaction implements ports.Transactor
func (a *TransactionServiceAdapter) ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error {
	if a.transactionManager == nil {
		// No transaction manager available - execute without transaction
		return operation(ctx)
	}

	// Convert to infrastructure call with default options
	options := interfaces.DefaultTransactionOptions()
	return a.transactionManager.RunInTransactionWithOptions(ctx, options, operation)
}

// SupportsTransactions implements ports.Transactor
func (a *TransactionServiceAdapter) SupportsTransactions() bool {
	return a.transactionManager != nil && a.transactionManager.SupportsTransactions()
}

// IsTransactionActive implements ports.Transactor
func (a *TransactionServiceAdapter) IsTransactionActive(ctx context.Context) bool {
	if a.transactionManager == nil {
		return false
	}

	_, hasTransaction := a.transactionManager.GetTransaction(ctx)
	return hasTransaction
}
