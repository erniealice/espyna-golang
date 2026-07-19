//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"testing"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
)

// FIX-5 DB-gated ROLLBACK-ONLY proof for the adapter leg of the graph-wide spawn
// pre-lock: LockTemplatePhasesForSpawn (the method preLockSpawnGraph hands the
// whole deduped graph to) — (a) FAILS CLOSED outside a transaction, and (b) inside
// a transaction actually acquires FOR UPDATE row locks on the given
// job_template_phase parents (a second session's FOR UPDATE on one of them blocks
// and times out), regardless of the order the ids were supplied in (the adapter
// sorts + issues one `ORDER BY id FOR UPDATE`).
func TestIntegration_SpawnPreLock_SortedForUpdateAcquisition(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Pick two real template-phase parents (any two).
	rows, err := db.Query(`SELECT id FROM job_template_phase WHERE active = true ORDER BY id LIMIT 2`)
	if err != nil {
		t.Fatalf("pick template phases: %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < 2 {
		t.Skip("need >= 2 job_template_phase rows")
	}

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresJobPhaseRepository(wsOps, "job_phase").(*PostgresJobPhaseRepository)

	// (a) Fail-closed outside a transaction: the pre-lock demands the ambient tx.
	if err := repo.LockTemplatePhasesForSpawn(context.Background(), ids); err == nil {
		t.Fatal("LockTemplatePhasesForSpawn must fail closed outside a transaction")
	}

	// (b) Inside a rolled-back tx, hand the ids in REVERSE order — the adapter
	// sorts and locks them FOR UPDATE; a second session must block on one.
	rollback := fmt.Errorf("fixp2b: intentional rollback")
	err = tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		reversed := []string{ids[1], ids[0]}
		if lerr := repo.LockTemplatePhasesForSpawn(txCtx, reversed); lerr != nil {
			return fmt.Errorf("in-tx pre-lock: %w", lerr)
		}

		// Second session: FOR UPDATE on the first parent must block + time out.
		tx2, berr := db.BeginTx(txCtx, nil)
		if berr != nil {
			return berr
		}
		defer tx2.Rollback()
		if _, serr := tx2.ExecContext(txCtx, "SET LOCAL lock_timeout = '600ms'"); serr != nil {
			return serr
		}
		start := time.Now()
		var got string
		perr := tx2.QueryRowContext(txCtx,
			`SELECT id FROM job_template_phase WHERE id = $1 FOR UPDATE`, ids[0]).Scan(&got)
		if perr == nil {
			return fmt.Errorf("second session acquired a parent the spawn pre-lock should hold — FOR UPDATE not taken")
		}
		t.Logf("SPAWN-PRELOCK: second session correctly blocked on the held parent after %s: %v",
			time.Since(start).Round(time.Millisecond), perr)
		return rollback
	})
	if err != rollback {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}
}
