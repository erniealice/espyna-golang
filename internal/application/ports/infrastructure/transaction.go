package infrastructure

import "context"

// Transactor provides transaction capabilities to use cases
// This is completely technology-agnostic - use cases don't know about databases
type Transactor interface {
	// ExecuteInTransaction executes a function within a transaction context
	// If transaction fails, the function's changes are rolled back
	ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error

	// SupportsTransactions returns true if this service can provide transactions
	SupportsTransactions() bool

	// IsTransactionActive returns true if the context has an active transaction
	IsTransactionActive(ctx context.Context) bool
}

// NoOpTransactor does nothing - used as fallback when transactions unavailable
type NoOpTransactor struct{}

// NewNoOpTransactor creates a no-operation transaction service
func NewNoOpTransactor() Transactor {
	return &NoOpTransactor{}
}

// ExecuteInTransaction implements Transactor - just executes the operation
func (s *NoOpTransactor) ExecuteInTransaction(ctx context.Context, operation func(ctx context.Context) error) error {
	return operation(ctx)
}

// SupportsTransactions implements Transactor - always returns false
func (s *NoOpTransactor) SupportsTransactions() bool {
	return false
}

// IsTransactionActive implements Transactor - always returns false
func (s *NoOpTransactor) IsTransactionActive(ctx context.Context) bool {
	return false
}
