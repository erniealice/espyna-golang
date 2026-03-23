package entity

import (
	"context"
	"database/sql"
)

// dbExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
// Mirrors the interface in the core package (copied to avoid circular imports).
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// executorProvider provides a transaction-aware database executor.
// PostgresOperations in the core package satisfies this interface.
type executorProvider interface {
	GetExecutor(ctx context.Context) dbExecutor
}
