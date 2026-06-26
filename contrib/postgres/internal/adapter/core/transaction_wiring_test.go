//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	ports "github.com/erniealice/espyna-golang/internal/application/ports"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	txbridge "github.com/erniealice/espyna-golang/shared/database/transactions"
)

// TestWiredTransactorIsRealAndSupportsTransactions proves the boot wiring's
// payload: NewTransactionServiceAdapter(<postgres manager>) yields a non-nil
// ports.Transactor whose SupportsTransactions() is true. This is the exact value
// container.Initialize() assigns to services.Transaction, and the two casts
// (`services.Transaction.(ports.Transactor)`) consume — so a true here means the
// use-case `if Transactor != nil && Transactor.SupportsTransactions()` branches
// activate. Hermetic: NewPostgreSQLTransactionManager + the bridge never touch the
// DB; SupportsTransactions() is a pure capability flag.
func TestWiredTransactorIsRealAndSupportsTransactions(t *testing.T) {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	defer db.Close()

	mgr := NewPostgreSQLTransactionManager(db) // interfaces.TransactionManager
	if mgr == nil {
		t.Fatal("NewPostgreSQLTransactionManager returned nil")
	}

	var tx ports.Transactor = txbridge.NewTransactionServiceAdapter(mgr)
	if tx == nil {
		t.Fatal("NewTransactionServiceAdapter returned nil — the wired port would be dormant")
	}
	if !tx.SupportsTransactions() {
		t.Fatal("SupportsTransactions() == false — use cases would take the no-tx executeCore branch")
	}
}

// TestReEntrancyGuardJoinsActiveTx proves the manager-level re-entrancy guard:
// when an active (Pending) PostgreSQL transaction is already in ctx, a nested
// RunInTransactionWithOptions does NOT open a second physical transaction — it
// runs the closure directly on the SAME tx in ctx. Hermetic: we manufacture a
// Pending *PostgreSQLTransaction and seed ctx with it (no BeginTx), so if the
// guard ever regressed to calling StartTransactionWithOptions -> db.BeginTx, the
// nil/closed db would surface (and the asserted tx-identity below would break).
func TestReEntrancyGuardJoinsActiveTx(t *testing.T) {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	defer db.Close()

	tm := &PostgreSQLTransactionManager{db: db}

	// Manufacture an already-active outer tx and place it in ctx, exactly as the
	// outer RunInTransactionWithOptions would via operations.WithTransaction.
	outer := &PostgreSQLTransaction{
		state: interfaces.TransactionStatePending,
		db:    db,
		id:    "outer-tx",
	}
	ctx := operations.WithTransaction(context.Background(), outer)

	// (1) StartTransactionWithOptions must RETURN THE EXISTING tx (no BeginTx).
	got, err := tm.StartTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions())
	if err != nil {
		t.Fatalf("StartTransactionWithOptions errored on a nested call: %v", err)
	}
	if got.ID() != "outer-tx" {
		t.Fatalf("nested StartTransactionWithOptions returned a NEW tx (id=%q); expected to join the active outer tx (id=%q)", got.ID(), "outer-tx")
	}

	// (2) RunInTransactionWithOptions must run the closure directly on the active
	// tx: the tx visible inside fn is the SAME outer tx, and the manager does NOT
	// begin/commit/rollback a second physical tx.
	ran := false
	err = tm.RunInTransactionWithOptions(ctx, interfaces.DefaultTransactionOptions(), func(innerCtx context.Context) error {
		ran = true
		inner, ok := operations.GetTransactionFromContext(innerCtx)
		if !ok {
			t.Fatal("no transaction in ctx inside nested RunInTransactionWithOptions")
		}
		if inner.ID() != "outer-tx" {
			t.Fatalf("nested closure sees a DIFFERENT tx (id=%q); expected the active outer tx (id=%q) — a second BeginTx would deadlock/partial-commit", inner.ID(), "outer-tx")
		}
		// The outer tx must still be Pending (the guard must not have committed it).
		if outer.State() != interfaces.TransactionStatePending {
			t.Fatalf("outer tx state changed to %s during nested run; the inner call must not commit/rollback the joined tx", outer.State())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("nested RunInTransactionWithOptions errored: %v", err)
	}
	if !ran {
		t.Fatal("nested closure did not run")
	}
	// After a successful nested run, the outer tx must STILL be Pending — the
	// outermost owner (not this nested call) is responsible for commit.
	if outer.State() != interfaces.TransactionStatePending {
		t.Fatalf("outer tx state is %s after nested run; expected still Pending (nested call must not finalize the joined tx)", outer.State())
	}
}

// TestExecuteInTransactionMakesTxActiveInCtx is the end-to-end runtime proof
// through the SAME bridge the boot wiring uses: inside the operation passed to
// the application-layer Transactor, a transaction IS active in ctx (GetExecutor
// returns the *sql.Tx, GetTransactionFromContext finds a Pending tx). Requires a
// live DB (BeginTx) — self-skips without TEST_DATABASE_URL, mirroring the
// existing TestRollbackNoPersist.
func TestExecuteInTransactionMakesTxActiveInCtx(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	mgr := NewPostgreSQLTransactionManager(db)
	transactor := txbridge.NewTransactionServiceAdapter(mgr)
	ops := &PostgresOperations{db: db}

	ran := false
	err = transactor.ExecuteInTransaction(context.Background(), func(ctx context.Context) error {
		ran = true

		// (a) the bridge reports an active tx in this ctx
		if !transactor.IsTransactionActive(ctx) {
			t.Error("IsTransactionActive(ctx) == false inside ExecuteInTransaction")
		}
		// (b) a Pending tx is discoverable in ctx
		txn, ok := operations.GetTransactionFromContext(ctx)
		if !ok || txn.State() != interfaces.TransactionStatePending {
			t.Errorf("expected a Pending tx in ctx; ok=%v state=%v", ok, txn)
		}
		// (c) GetExecutor routes onto the *sql.Tx, NOT the *sql.DB
		exec := ops.getExecutor(ctx)
		if _, isTx := exec.(*sql.Tx); !isTx {
			t.Errorf("getExecutor returned %T inside tx; expected *sql.Tx", exec)
		}
		if exec == ops.db {
			t.Error("getExecutor returned the base *sql.DB inside a transaction")
		}

		// (d) re-entrancy end-to-end: a nested ExecuteInTransaction must join the
		// SAME tx (same *sql.Tx executor), not open a second one.
		return transactor.ExecuteInTransaction(ctx, func(nestedCtx context.Context) error {
			nestedTxn, ok := operations.GetTransactionFromContext(nestedCtx)
			if !ok {
				t.Error("nested ExecuteInTransaction: no tx in ctx")
				return nil
			}
			if nestedTxn.ID() != txn.ID() {
				t.Errorf("nested ExecuteInTransaction opened a NEW tx (id=%q); expected to join outer (id=%q)", nestedTxn.ID(), txn.ID())
			}
			if ops.getExecutor(nestedCtx).(*sql.Tx) != ops.getExecutor(ctx).(*sql.Tx) {
				t.Error("nested executor is a different *sql.Tx — second physical transaction leaked")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("ExecuteInTransaction returned error: %v", err)
	}
	if !ran {
		t.Fatal("operation did not run inside ExecuteInTransaction")
	}
}
