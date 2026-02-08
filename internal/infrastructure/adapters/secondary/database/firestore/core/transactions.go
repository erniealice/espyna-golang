//go:build firestore

package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
)

// FirestoreTransaction implements interfaces.Transaction for Firestore
type FirestoreTransaction struct {
	id      string
	ctx     context.Context
	client  *firestore.Client
	options interfaces.TransactionOptions
	state   interfaces.TransactionState
	tx      *firestore.Transaction
	mu      sync.RWMutex
}

// NewFirestoreTransaction creates a new Firestore transaction
func NewFirestoreTransaction(ctx context.Context, client *firestore.Client, options interfaces.TransactionOptions) *FirestoreTransaction {
	return &FirestoreTransaction{
		id:      fmt.Sprintf("firestore_tx_%d", time.Now().UnixNano()),
		ctx:     ctx,
		client:  client,
		options: options,
		state:   interfaces.TransactionStatePending,
	}
}

// Begin starts the Firestore transaction
func (ft *FirestoreTransaction) Begin(ctx context.Context) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.state != interfaces.TransactionStatePending {
		return fmt.Errorf("transaction is already in state %s", ft.state)
	}

	// For Firestore, we don't pre-create the transaction here
	// It will be created when actually needed in RunTransaction
	return nil
}

// Commit commits the Firestore transaction
func (ft *FirestoreTransaction) Commit(ctx context.Context) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.state != interfaces.TransactionStatePending {
		return fmt.Errorf("cannot commit transaction in state %s", ft.state)
	}

	// For Firestore, commit happens automatically when RunTransaction completes successfully
	ft.state = interfaces.TransactionStateCommitted
	return nil
}

// Rollback rolls back the Firestore transaction
func (ft *FirestoreTransaction) Rollback(ctx context.Context) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.state == interfaces.TransactionStateRolledBack {
		return nil // Already rolled back
	}

	if ft.state == interfaces.TransactionStateCommitted {
		return fmt.Errorf("cannot rollback committed transaction")
	}

	// For Firestore, rollback happens automatically when RunTransaction fails
	ft.state = interfaces.TransactionStateRolledBack
	return nil
}

// Context returns the transaction context
func (ft *FirestoreTransaction) Context() context.Context {
	return operations.WithTransaction(ft.ctx, ft)
}

// State returns the current transaction state
func (ft *FirestoreTransaction) State() interfaces.TransactionState {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.state
}

// ID returns the transaction identifier
func (ft *FirestoreTransaction) ID() string {
	return ft.id
}

// FirestoreTransactionManager implements interfaces.TransactionManager for Firestore
type FirestoreTransactionManager struct {
	client *firestore.Client
}

// NewFirestoreTransactionManager creates a new FirestoreTransactionManager
func NewFirestoreTransactionManager(client *firestore.Client) interfaces.TransactionManager {
	return &FirestoreTransactionManager{
		client: client,
	}
}

// StartTransaction creates and begins a new transaction
func (tm *FirestoreTransactionManager) StartTransaction(ctx context.Context) (interfaces.Transaction, error) {
	return tm.StartTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions())
}

// StartTransactionWithOptions creates and begins a new transaction with specific options
func (tm *FirestoreTransactionManager) StartTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions) (interfaces.Transaction, error) {
	if tm.client == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	tx := NewFirestoreTransaction(ctx, tm.client, options)
	if err := tx.Begin(ctx); err != nil {
		return nil, fmt.Errorf("failed to begin firestore transaction: %w", err)
	}

	return tx, nil
}

// RunInTransaction executes a function within a transaction, handling commit/rollback automatically
func (tm *FirestoreTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.RunInTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions(), fn)
}

// RunInTransactionWithOptions executes a function within a transaction with specific options
func (tm *FirestoreTransactionManager) RunInTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions, fn func(ctx context.Context) error) error {
	if tm.client == nil {
		return fmt.Errorf("firestore client is nil")
	}

	// Use Firestore's native RunTransaction which handles retries and commit/rollback
	return tm.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Create our transaction wrapper
		firestoreTx := NewFirestoreTransaction(ctx, tm.client, options)
		firestoreTx.tx = tx // Set the native Firestore transaction

		// Create context with our transaction
		txCtx := operations.WithTransaction(ctx, firestoreTx)

		// Execute the function
		if err := fn(txCtx); err != nil {
			// Mark transaction as rolled back
			firestoreTx.mu.Lock()
			firestoreTx.state = interfaces.TransactionStateRolledBack
			firestoreTx.mu.Unlock()
			return err
		}

		// Mark transaction as committed
		firestoreTx.mu.Lock()
		firestoreTx.state = interfaces.TransactionStateCommitted
		firestoreTx.mu.Unlock()

		return nil
	})
}

// GetTransaction retrieves the current transaction from context, if any
func (tm *FirestoreTransactionManager) GetTransaction(ctx context.Context) (interfaces.Transaction, bool) {
	return operations.GetTransactionFromContext(ctx)
}

// SupportsTransactions returns true if the underlying database supports transactions
func (tm *FirestoreTransactionManager) SupportsTransactions() bool {
	return true
}
