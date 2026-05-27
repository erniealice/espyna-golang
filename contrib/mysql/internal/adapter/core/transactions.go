//go:build mysql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
)

// MySQLTransactionManager implements interfaces.TransactionManager for MySQL.
type MySQLTransactionManager struct {
	db *sql.DB
}

// NewMySQLTransactionManager creates a new MySQLTransactionManager.
func NewMySQLTransactionManager(db *sql.DB) interfaces.TransactionManager {
	return &MySQLTransactionManager{db: db}
}

// MySQLTransaction implements interfaces.Transaction for MySQL.
type MySQLTransaction struct {
	tx    *sql.Tx
	db    *sql.DB
	ctx   context.Context
	state interfaces.TransactionState
	id    string
}

// StartTransaction creates and begins a new transaction.
func (tm *MySQLTransactionManager) StartTransaction(ctx context.Context) (interfaces.Transaction, error) {
	return tm.StartTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions())
}

// StartTransactionWithOptions creates and begins a new transaction with options.
func (tm *MySQLTransactionManager) StartTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions) (interfaces.Transaction, error) {
	txCtx := ctx
	if options.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Millisecond)
		_ = cancel
		txCtx = timeoutCtx
	}

	tx, err := tm.db.BeginTx(txCtx, &sql.TxOptions{ReadOnly: options.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("failed to begin mysql transaction: %w", err)
	}

	return &MySQLTransaction{
		tx:    tx,
		db:    tm.db,
		ctx:   txCtx,
		state: interfaces.TransactionStatePending,
		id:    generateMySQLTransactionID(),
	}, nil
}

// RunInTransaction executes fn within a transaction, committing on success and
// rolling back on error.
func (tm *MySQLTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.RunInTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions(), fn)
}

// RunInTransactionWithOptions executes fn within a transaction with options.
func (tm *MySQLTransactionManager) RunInTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions, fn func(ctx context.Context) error) error {
	tx, err := tm.StartTransactionWithOptions(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	txCtx := operations.WithTransaction(ctx, tx)

	var fnErr error
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
		if fnErr != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				fnErr = fmt.Errorf("transaction failed and rollback failed: %w (rollback: %v)", fnErr, rbErr)
			}
		} else {
			if cmErr := tx.Commit(ctx); cmErr != nil {
				fnErr = fmt.Errorf("transaction succeeded but commit failed: %w", cmErr)
			}
		}
	}()

	fnErr = fn(txCtx)
	return fnErr
}

// GetTransaction retrieves the current transaction from context, if any.
func (tm *MySQLTransactionManager) GetTransaction(ctx context.Context) (interfaces.Transaction, bool) {
	return operations.GetTransactionFromContext(ctx)
}

// SupportsTransactions returns true.
func (tm *MySQLTransactionManager) SupportsTransactions() bool { return true }

// ── MySQLTransaction methods ─────────────────────────────────────────────────

// Begin is a no-op for MySQL: BeginTx already started the transaction.
func (t *MySQLTransaction) Begin(ctx context.Context) error {
	if t.state != interfaces.TransactionStatePending {
		return fmt.Errorf("mysql transaction is not pending (state: %s)", t.state.String())
	}
	return nil
}

// Commit commits the transaction.
func (t *MySQLTransaction) Commit(ctx context.Context) error {
	if t.state != interfaces.TransactionStatePending {
		return fmt.Errorf("cannot commit mysql transaction in state: %s", t.state.String())
	}
	if err := t.tx.Commit(); err != nil {
		t.state = interfaces.TransactionStateRolledBack
		return fmt.Errorf("failed to commit mysql transaction: %w", err)
	}
	t.state = interfaces.TransactionStateCommitted
	return nil
}

// Rollback rolls back the transaction. Safe to call multiple times.
func (t *MySQLTransaction) Rollback(ctx context.Context) error {
	if t.state == interfaces.TransactionStateCommitted {
		return fmt.Errorf("cannot rollback committed mysql transaction")
	}
	if t.state == interfaces.TransactionStateRolledBack {
		return nil
	}
	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback mysql transaction: %w", err)
	}
	t.state = interfaces.TransactionStateRolledBack
	return nil
}

// Context returns the context associated with this transaction.
func (t *MySQLTransaction) Context() context.Context { return t.ctx }

// State returns the current state of the transaction.
func (t *MySQLTransaction) State() interfaces.TransactionState { return t.state }

// GetTx returns the underlying *sql.Tx.
func (t *MySQLTransaction) GetTx() *sql.Tx { return t.tx }

// ID returns the unique identifier for this transaction.
func (t *MySQLTransaction) ID() string { return t.id }

// generateMySQLTransactionID creates a unique transaction identifier.
func generateMySQLTransactionID() string {
	return fmt.Sprintf("mysql_tx_%d", time.Now().UnixNano())
}
