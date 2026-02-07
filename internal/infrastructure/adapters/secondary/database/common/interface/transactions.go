package interfaces

import (
	"context"
)

// TransactionState represents the current state of a transaction
type TransactionState int

const (
	// TransactionStatePending indicates the transaction has begun but not committed or rolled back
	TransactionStatePending TransactionState = iota
	// TransactionStateCommitted indicates the transaction has been successfully committed
	TransactionStateCommitted
	// TransactionStateRolledBack indicates the transaction has been rolled back
	TransactionStateRolledBack
)

// String returns the string representation of TransactionState
func (ts TransactionState) String() string {
	switch ts {
	case TransactionStatePending:
		return "pending"
	case TransactionStateCommitted:
		return "committed"
	case TransactionStateRolledBack:
		return "rolled_back"
	default:
		return "unknown"
	}
}

// Transaction represents a database transaction with lifecycle management
type Transaction interface {
	// Begin starts the transaction - may be called multiple times safely
	Begin(ctx context.Context) error

	// Commit commits the transaction
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction - safe to call multiple times
	Rollback(ctx context.Context) error

	// Context returns the context associated with this transaction
	Context() context.Context

	// State returns the current state of the transaction
	State() TransactionState

	// ID returns a unique identifier for this transaction
	ID() string
}

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	// ReadOnly indicates whether the transaction should be read-only
	ReadOnly bool

	// Timeout specifies the transaction timeout in milliseconds
	// A value of 0 or negative indicates no timeout
	Timeout int64
}

// DefaultTransactionOptions returns the default transaction options
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		ReadOnly: false,
		Timeout:  0, // No timeout by default
	}
}

// TransactionManager manages database transactions and provides transaction lifecycle operations
type TransactionManager interface {
	// StartTransaction creates and begins a new transaction
	StartTransaction(ctx context.Context) (Transaction, error)

	// StartTransactionWithOptions creates and begins a new transaction with specific options
	StartTransactionWithOptions(ctx context.Context, options TransactionOptions) (Transaction, error)

	// RunInTransaction executes a function within a transaction, handling commit/rollback automatically
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// RunInTransactionWithOptions executes a function within a transaction with specific options
	RunInTransactionWithOptions(ctx context.Context, options TransactionOptions, fn func(ctx context.Context) error) error

	// GetTransaction retrieves the current transaction from context, if any
	GetTransaction(ctx context.Context) (Transaction, bool)

	// SupportsTransactions returns true if the underlying database supports transactions
	SupportsTransactions() bool
}
