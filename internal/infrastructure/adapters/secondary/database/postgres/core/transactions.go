//go:build postgres

package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
)

// PostgreSQLTransactionManager implements interfaces.TransactionManager for PostgreSQL
type PostgreSQLTransactionManager struct {
	db *sql.DB
}

// NewPostgreSQLTransactionManager creates a new PostgreSQLTransactionManager
func NewPostgreSQLTransactionManager(db *sql.DB) interfaces.TransactionManager {
	return &PostgreSQLTransactionManager{
		db: db,
	}
}

// PostgreSQLTransaction implements interfaces.Transaction for PostgreSQL
type PostgreSQLTransaction struct {
	tx    *sql.Tx
	db    *sql.DB
	ctx   context.Context
	state interfaces.TransactionState
	id    string
}

// StartTransaction creates and begins a new transaction
func (tm *PostgreSQLTransactionManager) StartTransaction(ctx context.Context) (interfaces.Transaction, error) {
	return tm.StartTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions())
}

// StartTransactionWithOptions creates and begins a new transaction with specific options
func (tm *PostgreSQLTransactionManager) StartTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions) (interfaces.Transaction, error) {
	// Create transaction context with timeout if specified
	txCtx := ctx
	if options.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Millisecond)
		_ = cancel // We'll handle cleanup in the transaction lifecycle
		txCtx = timeoutCtx
	}

	// Begin the SQL transaction
	txOptions := &sql.TxOptions{
		ReadOnly: options.ReadOnly,
	}

	tx, err := tm.db.BeginTx(txCtx, txOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction instance
	postgresTx := &PostgreSQLTransaction{
		tx:    tx,
		db:    tm.db,
		ctx:   txCtx,
		state: interfaces.TransactionStatePending,
		id:    generateTransactionID(),
	}

	return postgresTx, nil
}

// RunInTransaction executes a function within a transaction, handling commit/rollback automatically
func (tm *PostgreSQLTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.RunInTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions(), fn)
}

// RunInTransactionWithOptions executes a function within a transaction with specific options
func (tm *PostgreSQLTransactionManager) RunInTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions, fn func(ctx context.Context) error) error {
	// Start transaction
	tx, err := tm.StartTransactionWithOptions(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Add transaction to context
	txCtx := operations.WithTransaction(ctx, tx)

	// Execute function with proper error handling
	var fnErr error
	defer func() {
		if r := recover(); r != nil {
			// Handle panic - rollback transaction
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// Log rollback error but don't override the original panic
				fmt.Printf("Failed to rollback transaction after panic: %v\n", rollbackErr)
			}
			panic(r) // Re-panic
		}

		if fnErr != nil {
			// Function returned error - rollback
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				fnErr = fmt.Errorf("transaction failed and rollback failed: %w (rollback error: %v)", fnErr, rollbackErr)
			}
		} else {
			// Function succeeded - commit
			if commitErr := tx.Commit(ctx); commitErr != nil {
				fnErr = fmt.Errorf("transaction function succeeded but commit failed: %w", commitErr)
			}
		}
	}()

	// Execute the function
	fnErr = fn(txCtx)
	return fnErr
}

// GetTransaction retrieves the current transaction from context, if any
func (tm *PostgreSQLTransactionManager) GetTransaction(ctx context.Context) (interfaces.Transaction, bool) {
	return operations.GetTransactionFromContext(ctx)
}

// SupportsTransactions returns true if the underlying database supports transactions
func (tm *PostgreSQLTransactionManager) SupportsTransactions() bool {
	return true
}

// PostgreSQLTransaction implementation

// Begin starts the transaction - may be called multiple times safely
func (pt *PostgreSQLTransaction) Begin(ctx context.Context) error {
	// PostgreSQL transaction is already begun in StartTransaction
	// This is a no-op for PostgreSQL since BeginTx already started it
	if pt.state != interfaces.TransactionStatePending {
		return fmt.Errorf("transaction is not in pending state, current state: %s", pt.state.String())
	}
	return nil
}

// Commit commits the transaction
func (pt *PostgreSQLTransaction) Commit(ctx context.Context) error {
	if pt.state != interfaces.TransactionStatePending {
		return fmt.Errorf("cannot commit transaction in state: %s", pt.state.String())
	}

	if pt.tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	err := pt.tx.Commit()
	if err != nil {
		pt.state = interfaces.TransactionStateRolledBack // Mark as failed
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	pt.state = interfaces.TransactionStateCommitted
	return nil
}

// Rollback rolls back the transaction - safe to call multiple times
func (pt *PostgreSQLTransaction) Rollback(ctx context.Context) error {
	if pt.state == interfaces.TransactionStateCommitted {
		return fmt.Errorf("cannot rollback committed transaction")
	}

	if pt.state == interfaces.TransactionStateRolledBack {
		// Already rolled back, this is safe to call multiple times
		return nil
	}

	if pt.tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	err := pt.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	pt.state = interfaces.TransactionStateRolledBack
	return nil
}

// Context returns the context associated with this transaction
func (pt *PostgreSQLTransaction) Context() context.Context {
	return pt.ctx
}

// State returns the current state of the transaction
func (pt *PostgreSQLTransaction) State() interfaces.TransactionState {
	return pt.state
}

// ID returns a unique identifier for this transaction
func (pt *PostgreSQLTransaction) ID() string {
	return pt.id
}

// generateTransactionID creates a unique transaction identifier
func generateTransactionID() string {
	return fmt.Sprintf("pg_tx_%d", time.Now().UnixNano())
}
