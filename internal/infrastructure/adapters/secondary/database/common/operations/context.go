package operations

import (
	"context"
	"fmt"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
)

// Context keys for transaction management
type contextKey string

const (
	// TransactionContextKey is used to store Transaction in context
	TransactionContextKey contextKey = "database_transaction"

	// TransactionManagerContextKey is used to store TransactionManager in context
	TransactionManagerContextKey contextKey = "transaction_manager"
)

// WithTransaction adds a transaction to the context
func WithTransaction(ctx context.Context, tx interfaces.Transaction) context.Context {
	return context.WithValue(ctx, TransactionContextKey, tx)
}

// GetTransactionFromContext retrieves a transaction from context
func GetTransactionFromContext(ctx context.Context) (interfaces.Transaction, bool) {
	tx, ok := ctx.Value(TransactionContextKey).(interfaces.Transaction)
	return tx, ok
}

// WithTransactionManager adds a transaction manager to the context
func WithTransactionManager(ctx context.Context, tm interfaces.TransactionManager) context.Context {
	return context.WithValue(ctx, TransactionManagerContextKey, tm)
}

// GetTransactionManagerFromContext retrieves a transaction manager from context
func GetTransactionManagerFromContext(ctx context.Context) (interfaces.TransactionManager, bool) {
	tm, ok := ctx.Value(TransactionManagerContextKey).(interfaces.TransactionManager)
	return tm, ok
}

// IsTransactionContext checks if the context contains an active transaction
func IsTransactionContext(ctx context.Context) bool {
	tx, exists := GetTransactionFromContext(ctx)
	if !exists {
		return false
	}
	return tx.State() == interfaces.TransactionStatePending
}

// generateTransactionID creates a unique transaction identifier
func generateTransactionID() string {
	return fmt.Sprintf("tx_%s", generateUUID())
}
