//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
	"github.com/erniealice/espyna-golang/schema"

	// fulfillment is the representative entity for the Phase-2 shadow-agreement
	// test: it carries all four column-kind edge cases (int64 audit-millis,
	// google.protobuf.Timestamp business timestamps, google.protobuf.Struct->jsonb
	// metadata, bool active) and zero (db).ignore *_string mirrors, so a true
	// descriptor-vs-reflection agreement is unambiguous. Importing the pb package
	// here guarantees its init() registers the message in protoregistry.GlobalTypes
	// BEFORE the test calls schema.Build() (Go runs imported package init()s before
	// any test function), so schema.Global is populated for the "fulfillment" table.
	fulfillmentv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
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

// --- Phase-2 shadow-agreement unit test -------------------------------------
//
// These tests assert the descriptor-driven SHADOW path agrees with a faithful
// reflection-derived view for a representative entity (fulfillment), and that the
// agreement metric ticks AGREE (not DISAGREE) on each dimension. They run with NO
// database — the shadow assertions are pure functions over the registry + the
// reflected column-set/types the caller passes in. This is the Q-RC4 0-disagreement
// observability the Phase-4 enforce flip is gated on.

// reflectedViewFromDescriptor builds the reflection-derived inputs (valid-column
// set + column-type map) that a CORRECT live schema would produce for tableName,
// sourced from the descriptor so the agreement scenario is exact. IsBigintMillis
// columns map to information_schema "bigint"; Timestamp columns map to "timestamp
// with time zone"; everything else to a benign non-bigint placeholder. This mirrors
// what getTableColumns / getTableColumnTypes return at runtime when proto and DB are
// in sync.
func reflectedViewFromDescriptor(t *testing.T, tableName string) (map[string]bool, map[string]string) {
	t.Helper()
	cols, ok := schema.ColsFor(tableName)
	if !ok {
		t.Fatalf("schema.ColsFor(%q) = false; registry not populated (schema.Build did not run or pb not imported)", tableName)
	}
	validColumns := make(map[string]bool, len(cols))
	columnTypes := make(map[string]string, len(cols))
	for _, c := range cols {
		validColumns[c.Name] = true
		switch {
		case c.IsBigintMillis:
			columnTypes[c.Name] = "bigint"
		case c.IsTimestamp:
			columnTypes[c.Name] = "timestamp with time zone"
		default:
			columnTypes[c.Name] = "text"
		}
	}
	return validColumns, columnTypes
}

func TestShadowAgreementFulfillment(t *testing.T) {
	// Touch the imported pb type so the import is unambiguously load-bearing and
	// fulfillment is registered before Build().
	if (&fulfillmentv1.Fulfillment{}).ProtoReflect().Descriptor() == nil {
		t.Fatal("fulfillment descriptor unexpectedly nil")
	}
	if err := schema.Build(); err != nil {
		t.Fatalf("schema.Build() failed: %v", err)
	}

	const table = "fulfillment"
	validColumns, columnTypes := reflectedViewFromDescriptor(t, table)

	a0, d0 := ShadowAgreementSnapshot()

	// (1) column-set: descriptor vs the faithful reflected set -> AGREE.
	shadowAssertColumnSet(table, validColumns)

	// (2) timestamp-type: both audit columns -> AGREE on the bigint axis.
	if got := shadowTimestampType(table, "date_created", columnTypes); got != "bigint" {
		t.Errorf("shadowTimestampType(date_created) = %q, want %q (reflection authoritative)", got, "bigint")
	}
	if got := shadowTimestampType(table, "date_modified", columnTypes); got != "bigint" {
		t.Errorf("shadowTimestampType(date_modified) = %q, want %q", got, "bigint")
	}

	// (3) timestamp-VALUE: descriptor-derived value equals reflected-derived value
	// for the same `now` -> AGREE (this is the Phase-2 gap-closing check).
	now := time.Now().UTC()
	shadowAssertAutoTimestamp(table, "date_created", columnTypes, now)
	shadowAssertAutoTimestamp(table, "date_modified", columnTypes, now)

	// (4) drop-set: a data map of only-valid columns drops nothing on either side
	// -> AGREE. (reflectedSkipped is empty: the reflected path would drop nothing.)
	data := map[string]any{"id": "x", "status": "PENDING", "active": true}
	shadowAssertDropSet(table, data, nil, false)

	a1, d1 := ShadowAgreementSnapshot()
	if d1 != d0 {
		t.Errorf("shadow DISAGREE count rose from %d to %d for a faithful-agreement scenario; want unchanged", d0, d1)
	}
	// 6 dimensions ticked AGREE: column-set, 2x timestamp-type, 2x timestamp-value, drop-set.
	if got, want := a1-a0, int64(6); got != want {
		t.Errorf("shadow AGREE count rose by %d, want %d (column-set + 2 ts-type + 2 ts-value + drop-set)", got, want)
	}
}

// TestShadowAutoTimestampValueDivergence proves the Phase-2 value-axis check
// actually fires a DISAGREE when the descriptor type and reflected type would
// produce different written values (bigint-millis int64 vs TIMESTAMPTZ time.Time)
// — without affecting any write (observe-only). It drives shadowAssertAutoTimestamp
// directly with a reflected-types map that disagrees with the descriptor.
func TestShadowAutoTimestampValueDivergence(t *testing.T) {
	if (&fulfillmentv1.Fulfillment{}).ProtoReflect().Descriptor() == nil {
		t.Fatal("fulfillment descriptor unexpectedly nil")
	}
	if err := schema.Build(); err != nil {
		t.Fatalf("schema.Build() failed: %v", err)
	}

	const table = "fulfillment"
	// date_created is descriptor bigint-millis; pretend the live column reflected as
	// a TIMESTAMPTZ (non-bigint) -> the written value would differ (int64 ms vs
	// time.Time) -> DISAGREE.
	reflectedTypes := map[string]string{"date_created": "timestamp with time zone"}

	_, d0 := ShadowAgreementSnapshot()
	shadowAssertAutoTimestamp(table, "date_created", reflectedTypes, time.Now().UTC())
	_, d1 := ShadowAgreementSnapshot()
	if d1 != d0+1 {
		t.Errorf("expected exactly 1 DISAGREE tick on a value divergence, got delta %d", d1-d0)
	}
}
