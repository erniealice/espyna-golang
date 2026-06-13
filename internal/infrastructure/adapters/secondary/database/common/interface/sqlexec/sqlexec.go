package sqlexec

import (
	"context"
	"database/sql"
)

// DBExecutor abstracts *sql.DB and *sql.Tx for uniform query execution.
// Using this shared interface avoids the "missing method GetExecutor" panic
// that occurs when adapter packages each define their own unexported copy.
//
// This package is intentionally separate from the dialect-neutral
// database/interfaces package so that non-SQL adapters (Firestore, mock)
// do not transitively depend on database/sql.
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
