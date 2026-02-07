//go:build mock_db

package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/model"
)

// MockTransaction implements Transaction interface for testing
type MockTransaction struct {
	id            string
	state         interfaces.TransactionState
	ctx           context.Context
	operations    []MockOperation
	shouldFail    map[string]bool  // operation -> should fail
	failureErrors map[string]error // operation -> error to return
	mu            sync.RWMutex
}

// MockOperation represents an operation performed in a mock transaction
type MockOperation struct {
	Type       string
	Collection string
	DocumentID string
	Data       map[string]any
	Timestamp  int64
}

// NewMockTransaction creates a new mock transaction
func NewMockTransaction(ctx context.Context) *MockTransaction {
	return &MockTransaction{
		id:            generateTransactionID(),
		state:         interfaces.TransactionStatePending,
		ctx:           ctx,
		operations:    make([]MockOperation, 0),
		shouldFail:    make(map[string]bool),
		failureErrors: make(map[string]error),
	}
}

// Begin starts the mock transaction
func (mt *MockTransaction) Begin(ctx context.Context) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.shouldFail["begin"] {
		if err, exists := mt.failureErrors["begin"]; exists {
			return err
		}
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeBeginFailed,
			"mock transaction begin failed",
			"begin",
			mt.id,
		)
	}

	if mt.state != interfaces.TransactionStatePending {
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeInvalidState,
			fmt.Sprintf("transaction is already in state %s", mt.state),
			"begin",
			mt.id,
		)
	}

	return nil
}

// Commit commits the mock transaction
func (mt *MockTransaction) Commit(ctx context.Context) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.shouldFail["commit"] {
		mt.state = interfaces.TransactionStateRolledBack
		if err, exists := mt.failureErrors["commit"]; exists {
			return err
		}
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeCommitFailed,
			"mock transaction commit failed",
			"commit",
			mt.id,
		)
	}

	if mt.state != interfaces.TransactionStatePending {
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeInvalidState,
			fmt.Sprintf("cannot commit transaction in state %s", mt.state),
			"commit",
			mt.id,
		)
	}

	mt.state = interfaces.TransactionStateCommitted
	return nil
}

// Rollback rolls back the mock transaction
func (mt *MockTransaction) Rollback(ctx context.Context) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.shouldFail["rollback"] {
		if err, exists := mt.failureErrors["rollback"]; exists {
			return err
		}
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeRollbackFailed,
			"mock transaction rollback failed",
			"rollback",
			mt.id,
		)
	}

	if mt.state == interfaces.TransactionStateRolledBack {
		return nil // Already rolled back
	}

	if mt.state == interfaces.TransactionStateCommitted {
		return model.NewTransactionErrorWithID(
			model.TransactionErrorCodeInvalidState,
			"cannot rollback committed transaction",
			"rollback",
			mt.id,
		)
	}

	mt.state = interfaces.TransactionStateRolledBack
	mt.operations = nil // Clear operations on rollback
	return nil
}

// Context returns the transaction context
func (mt *MockTransaction) Context() context.Context {
	return context.WithValue(mt.ctx, "mock_transaction", mt)
}

// State returns the current transaction state
func (mt *MockTransaction) State() interfaces.TransactionState {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.state
}

// ID returns the transaction identifier
func (mt *MockTransaction) ID() string {
	return mt.id
}

// MockTransactionManager implements TransactionManager for testing
type MockTransactionManager struct {
	shouldFailStart     bool
	shouldFailRunInTx   bool
	startFailureError   error
	runInTxFailureError error
	createdTransactions []*MockTransaction
	mu                  sync.RWMutex
}

// NewMockTransactionManager creates a new mock transaction manager
func NewMockTransactionManager() interfaces.TransactionManager {
	return &MockTransactionManager{
		createdTransactions: make([]*MockTransaction, 0),
	}
}

// StartTransaction creates a new mock transaction
func (mtm *MockTransactionManager) StartTransaction(ctx context.Context) (interfaces.Transaction, error) {
	return mtm.StartTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions())
}

// StartTransactionWithOptions creates a new mock transaction with options
func (mtm *MockTransactionManager) StartTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions) (interfaces.Transaction, error) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if mtm.shouldFailStart {
		if mtm.startFailureError != nil {
			return nil, mtm.startFailureError
		}
		return nil, model.NewTransactionError(
			model.TransactionErrorCodeBeginFailed,
			"mock transaction manager start failed",
			"start_transaction",
		)
	}

	tx := NewMockTransaction(ctx)
	mtm.createdTransactions = append(mtm.createdTransactions, tx)

	// Begin the transaction
	if err := tx.Begin(ctx); err != nil {
		return nil, err
	}

	return tx, nil
}

// RunInTransaction executes a function within a mock transaction with retry logic
func (mtm *MockTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return mtm.RunInTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions(), fn)
}

// RunInTransactionWithOptions executes a function within a mock transaction with options
func (mtm *MockTransactionManager) RunInTransactionWithOptions(ctx context.Context, options interfaces.TransactionOptions, fn func(ctx context.Context) error) error {
	if mtm.shouldFailRunInTx {
		if mtm.runInTxFailureError != nil {
			return mtm.runInTxFailureError
		}
		return model.NewTransactionError(
			model.TransactionErrorCodeGeneral,
			"mock run in transaction failed",
			"run_in_transaction",
		)
	}

	maxRetries := 3
	errorHandler := model.NewTransactionErrorHandler()
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		tx, err := mtm.StartTransaction(ctx)
		if err != nil {
			lastErr = err
			if !errorHandler.ShouldRetry(err, attempt, maxRetries) {
				break
			}
			continue
		}

		txCtx := context.WithValue(ctx, "mock_transaction", tx)

		fnErr := fn(txCtx)
		if fnErr != nil {
			// Function failed, rollback the transaction
			if rbErr := tx.Rollback(txCtx); rbErr != nil {
				lastErr = fmt.Errorf("function failed and rollback failed: %w (rollback: %v)", fnErr, rbErr)
			} else {
				lastErr = fnErr
			}

			// Check if we should retry
			if !errorHandler.ShouldRetry(lastErr, attempt, maxRetries) {
				break
			}
			continue
		}

		// Function succeeded, commit the transaction
		if commitErr := tx.Commit(txCtx); commitErr != nil {
			lastErr = commitErr
			if !errorHandler.ShouldRetry(commitErr, attempt, maxRetries) {
				break
			}
			continue
		}

		// Success!
		return nil
	}

	// All attempts failed
	return lastErr
}

// GetTransaction retrieves the current transaction from context
func (mtm *MockTransactionManager) GetTransaction(ctx context.Context) (interfaces.Transaction, bool) {
	tx, ok := ctx.Value("mock_transaction").(interfaces.Transaction)
	return tx, ok
}

// SupportsTransactions returns true for mock manager (configurable)
func (mtm *MockTransactionManager) SupportsTransactions() bool {
	return true
}

// SetShouldFailStart configures whether StartTransaction operations should fail
func (mtm *MockTransactionManager) SetShouldFailStart(shouldFail bool) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()
	mtm.shouldFailStart = shouldFail
}

// SetShouldFailRunInTx configures whether RunInTransaction operations should fail
func (mtm *MockTransactionManager) SetShouldFailRunInTx(shouldFail bool) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()
	mtm.shouldFailRunInTx = shouldFail
}

// SetStartFailureError configures custom error for StartTransaction failures
func (mtm *MockTransactionManager) SetStartFailureError(err error) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()
	mtm.startFailureError = err
}

// SetRunInTxFailureError configures custom error for RunInTransaction failures
func (mtm *MockTransactionManager) SetRunInTxFailureError(err error) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()
	mtm.runInTxFailureError = err
}

// generateTransactionID creates a unique transaction identifier
func generateTransactionID() string {
	return fmt.Sprintf("tx_mock_%d", time.Now().UnixNano())
}

// MockTransactionServiceAdapter adapts MockTransactionManager to ports.TransactionService
// This provides a clean bridge between infrastructure mocks and application ports
type MockTransactionServiceAdapter struct {
	mockTxManager        *MockTransactionManager
	supportsTransactions bool
}

// NewMockTransactionService creates a transaction service using infrastructure mock
func NewMockTransactionService(supportsTransactions bool) ports.TransactionService {
	if !supportsTransactions {
		return ports.NewNoOpTransactionService()
	}

	// Create infrastructure mock and cast to access setter methods
	txManager := NewMockTransactionManager().(*MockTransactionManager)

	return &MockTransactionServiceAdapter{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

// NewFailingMockTransactionService creates a transaction service that will fail RunInTransaction
func NewFailingMockTransactionService() ports.TransactionService {
	txManager := NewMockTransactionManager().(*MockTransactionManager)

	// Configure to fail at RunInTransaction level using setter method
	txManager.SetShouldFailRunInTx(true)

	return &MockTransactionServiceAdapter{
		mockTxManager:        txManager,
		supportsTransactions: true,
	}
}

// SupportsTransactions returns whether transactions are supported
func (m *MockTransactionServiceAdapter) SupportsTransactions() bool {
	return m.supportsTransactions
}

// IsTransactionActive checks if a transaction is currently active in the context
func (m *MockTransactionServiceAdapter) IsTransactionActive(ctx context.Context) bool {
	if !m.supportsTransactions {
		return false
	}
	_, inTx := m.mockTxManager.GetTransaction(ctx)
	return inTx
}

// ExecuteInTransaction executes the given function within a transaction
func (m *MockTransactionServiceAdapter) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if !m.supportsTransactions {
		return fn(ctx)
	}
	return m.mockTxManager.RunInTransaction(ctx, fn)
}

// GetMockTransactionManager returns the underlying infrastructure mock for advanced configuration
func (m *MockTransactionServiceAdapter) GetMockTransactionManager() *MockTransactionManager {
	return m.mockTxManager
}
