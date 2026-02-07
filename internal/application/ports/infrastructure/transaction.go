package infrastructure

import "context"

// TransactionService provides transaction capabilities to use cases
// This is completely technology-agnostic - use cases don't know about databases
type TransactionService interface {
	// ExecuteInTransaction executes a function within a transaction context
	// If transaction fails, the function's changes are rolled back
	ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error

	// SupportsTransactions returns true if this service can provide transactions
	SupportsTransactions() bool

	// IsTransactionActive returns true if the context has an active transaction
	IsTransactionActive(ctx context.Context) bool
}

// NoOpTransactionService does nothing - used as fallback when transactions unavailable
type NoOpTransactionService struct{}

// NewNoOpTransactionService creates a no-operation transaction service
func NewNoOpTransactionService() TransactionService {
	return &NoOpTransactionService{}
}

// ExecuteInTransaction implements TransactionService - just executes the operation
func (s *NoOpTransactionService) ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error {
	return operation(ctx)
}

// SupportsTransactions implements TransactionService - always returns false
func (s *NoOpTransactionService) SupportsTransactions() bool {
	return false
}

// IsTransactionActive implements TransactionService - always returns false
func (s *NoOpTransactionService) IsTransactionActive(ctx context.Context) bool {
	return false
}
