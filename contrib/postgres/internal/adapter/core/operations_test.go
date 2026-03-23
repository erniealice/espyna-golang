//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
)

func TestGetExecutorReturnsTxWhenTransactionActive(t *testing.T) {
	// Create a *sql.DB (doesn't need to actually connect)
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	defer db.Close()
	p := &PostgresOperations{db: db}

	// Create a PostgreSQLTransaction with Pending state
	// (same package — can access unexported fields directly)
	mockTx := &PostgreSQLTransaction{
		state: interfaces.TransactionStatePending,
		id:    "test-tx-1",
		// tx is nil — but getExecutor should still return it (not p.db)
	}

	// Store transaction in context
	ctx := operations.WithTransaction(context.Background(), mockTx)
	exec := p.getExecutor(ctx)

	// The executor should NOT be p.db when a pending tx is in context
	if exec == p.db {
		t.Error("expected getExecutor to return transaction executor, not p.db")
	}
}

func TestGetExecutorReturnsDatabaseWhenNoTransaction(t *testing.T) {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	defer db.Close()
	p := &PostgresOperations{db: db}

	// Plain context with no transaction
	ctx := context.Background()
	exec := p.getExecutor(ctx)

	// Should return p.db when no transaction in context
	if exec != p.db {
		t.Error("expected getExecutor to return p.db when no transaction in context")
	}
}

func TestGetExecutorReturnsDatabaseWhenTransactionNotPending(t *testing.T) {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	defer db.Close()
	p := &PostgresOperations{db: db}

	// Transaction in committed state (not pending)
	committedTx := &PostgreSQLTransaction{
		state: interfaces.TransactionStateCommitted,
		id:    "test-tx-committed",
	}

	ctx := operations.WithTransaction(context.Background(), committedTx)
	exec := p.getExecutor(ctx)

	// Should fall back to p.db since tx is not pending
	if exec != p.db {
		t.Error("expected getExecutor to return p.db when transaction is committed (not pending)")
	}
}

func TestRollbackNoPersist(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS _test_tx_table (id TEXT PRIMARY KEY, name TEXT, active BOOL DEFAULT true, date_created TIMESTAMPTZ DEFAULT NOW(), date_modified TIMESTAMPTZ DEFAULT NOW())`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer db.Exec(`DROP TABLE IF EXISTS _test_tx_table`)

	txManager := NewPostgreSQLTransactionManager(db)
	ops := &PostgresOperations{db: db}

	// Execute transaction that returns error → triggers rollback
	sentinelErr := fmt.Errorf("intentional rollback")
	err = txManager.RunInTransaction(context.Background(), func(ctx context.Context) error {
		_, createErr := ops.Create(ctx, "_test_tx_table", map[string]any{
			"id":   "test-rollback-id",
			"name": "should not persist",
		})
		if createErr != nil {
			return fmt.Errorf("create failed inside tx: %w", createErr)
		}
		return sentinelErr // force rollback
	})

	// err should be sentinelErr (transaction rolled back)
	if err == nil {
		t.Fatal("expected error from RunInTransaction")
	}

	// Assert no row persisted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM _test_tx_table WHERE id = 'test-rollback-id'`).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}
