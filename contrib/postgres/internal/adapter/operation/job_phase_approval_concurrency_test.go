//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// This is a REAL, DB-gated two-session concurrency barrier test (plan §P2 exit
// gate). It runs ONLY when TEST_DATABASE_URL points at a live database and it is
// ROLLBACK-ONLY: every transaction is rolled back, so it takes brief FOR UPDATE
// locks on live rows but mutates nothing. It exercises the exact lock protocol
// the transition adapter uses (jobPhaseParentLockSQL + jobPhaseSheetLockSQL) AND
// the REAL production cell-write lock function guardCellWrite (the exact function
// the task_outcome + job_task use cases now call before every leaf write — no
// longer a hand-rolled SELECT), to prove:
//   - two bulk transitions over overlapping/same sheets serialize (no lost flip)
//     and do NOT deadlock under the parent mutex + FOR UPDATE OF jp;
//   - a cell-write (a phase-row FOR UPDATE) racing a held transition blocks,
//     i.e. the transition's jp locks are respected;
//   - the PRODUCTION guardCellWrite (parent FOR SHARE → job_phase FOR UPDATE →
//     recheck) blocks under a held transition — the real writer protocol, not a
//     synthetic proxy (codex §1 CONFIRMED: the old synthetic test never called a
//     production mutation path).

type sheetFixture struct {
	templateID string
	phaseID    string
	workspace  string
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping DB-gated concurrency test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.Ping(); err != nil {
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}
	return db
}

func loadTwoSheets(t *testing.T, db *sql.DB) (a, b sheetFixture) {
	t.Helper()
	const q = `
		SELECT j.job_template_id, jp.template_phase_id, j.workspace_id
		FROM ` + "job_phase" + ` jp JOIN ` + "job" + ` j ON j.id = jp.job_id
		WHERE jp.active = true AND jp.template_phase_id IS NOT NULL
		  AND j.job_template_id IS NOT NULL AND j.workspace_id IS NOT NULL
		GROUP BY j.job_template_id, jp.template_phase_id, j.workspace_id
		HAVING count(*) >= 2
		ORDER BY count(*) DESC
		LIMIT 2`
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("load sheets: %v", err)
	}
	defer rows.Close()
	var fx []sheetFixture
	for rows.Next() {
		var s sheetFixture
		if err := rows.Scan(&s.templateID, &s.phaseID, &s.workspace); err != nil {
			t.Fatalf("scan sheet: %v", err)
		}
		fx = append(fx, s)
	}
	if len(fx) < 2 {
		t.Skip("need >= 2 distinct sheets with >= 2 phases; not present")
	}
	return fx[0], fx[1]
}

// lockSheet takes the parent mutex then the full sheet FOR UPDATE OF jp inside tx,
// mirroring lockParentAndSheet. It respects lockTimeout via SET LOCAL.
func lockSheet(ctx context.Context, tx *sql.Tx, s sheetFixture, lockTimeout string) error {
	if lockTimeout != "" {
		if _, err := tx.ExecContext(ctx, "SET LOCAL lock_timeout = '"+lockTimeout+"'"); err != nil {
			return err
		}
	}
	var parent string
	if err := tx.QueryRowContext(ctx, jobPhaseParentLockSQL, s.phaseID, s.templateID, s.workspace).Scan(&parent); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, jobPhaseSheetLockSQL(""), s.templateID, s.phaseID, s.workspace)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, job, st string
		if err := rows.Scan(&id, &job, &st); err != nil {
			return err
		}
	}
	return rows.Err()
}

// TestConcurrency_SameSheetSerializes proves the second transition on the SAME
// sheet BLOCKS on the first's FOR UPDATE OF jp locks (surfaced as a lock timeout).
func TestConcurrency_SameSheetSerializes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	a, _ := loadTwoSheets(t, db)
	ctx := context.Background()

	tx1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}
	defer tx1.Rollback()
	if err := lockSheet(ctx, tx1, a, ""); err != nil {
		t.Fatalf("tx1 lock sheet: %v", err)
	}

	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	defer tx2.Rollback()
	start := time.Now()
	err = lockSheet(ctx, tx2, a, "600ms")
	if err == nil {
		t.Fatal("tx2 acquired the same-sheet lock while tx1 held it — NOT serialized")
	}
	t.Logf("SAME-SHEET: tx2 correctly blocked and timed out after %s: %v", time.Since(start).Round(time.Millisecond), err)
}

// TestConcurrency_DifferentSheetsNoDeadlock proves two transitions over DIFFERENT
// sheets, acquired in opposite order, both succeed with no deadlock (the parent
// mutex + FOR UPDATE OF jp never row-marks the shared job rows).
func TestConcurrency_DifferentSheetsNoDeadlock(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	a, b := loadTwoSheets(t, db)
	ctx := context.Background()

	tx1, _ := db.BeginTx(ctx, nil)
	defer tx1.Rollback()
	tx2, _ := db.BeginTx(ctx, nil)
	defer tx2.Rollback()

	errc := make(chan error, 2)
	go func() { errc <- lockSheet(ctx, tx1, a, "3s") }() // A then (implicitly) done
	go func() { errc <- lockSheet(ctx, tx2, b, "3s") }() // B concurrently

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("different-sheet lock failed (possible deadlock): %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("different-sheet locks did not complete in 10s — suspected deadlock")
		}
	}
	t.Log("DIFFERENT-SHEET: both transitions locked their disjoint sheets concurrently — no deadlock")
}

// TestConcurrency_CellWriteBlockedByTransition proves a concurrent phase-row
// FOR UPDATE (a cell-write acquiring its job_phase row lock) BLOCKS while a
// transition holds the sheet's FOR UPDATE OF jp locks.
func TestConcurrency_CellWriteBlockedByTransition(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	a, _ := loadTwoSheets(t, db)
	ctx := context.Background()

	// pick one phase id in sheet a
	var phaseRowID string
	if err := db.QueryRowContext(ctx, jobPhaseSheetLockSQL("")+" LIMIT 1", a.templateID, a.phaseID, a.workspace).Scan(&phaseRowID, new(string), new(string)); err != nil {
		// jobPhaseSheetLockSQL already ends with FOR UPDATE OF jp; append LIMIT is
		// invalid after FOR UPDATE. Fall back to a plain id lookup.
		if err := db.QueryRowContext(ctx,
			`SELECT jp.id FROM job_phase jp JOIN job j ON j.id=jp.job_id WHERE jp.template_phase_id=$2 AND j.job_template_id=$1 AND j.workspace_id=$3 AND jp.active=true ORDER BY jp.id LIMIT 1`,
			a.templateID, a.phaseID, a.workspace).Scan(&phaseRowID); err != nil {
			t.Fatalf("pick phase id: %v", err)
		}
	}

	tx1, _ := db.BeginTx(ctx, nil)
	defer tx1.Rollback()
	if err := lockSheet(ctx, tx1, a, ""); err != nil {
		t.Fatalf("tx1 lock sheet: %v", err)
	}

	tx2, _ := db.BeginTx(ctx, nil)
	defer tx2.Rollback()
	if _, err := tx2.ExecContext(ctx, "SET LOCAL lock_timeout = '600ms'"); err != nil {
		t.Fatalf("set lock_timeout: %v", err)
	}
	var got string
	err := tx2.QueryRowContext(ctx, `SELECT id FROM job_phase WHERE id = $1 FOR UPDATE`, phaseRowID).Scan(&got)
	if err == nil {
		t.Fatal("cell-write acquired a phase-row lock the transition held — locks NOT respected")
	}
	t.Logf("CELL-WRITE: a concurrent phase-row FOR UPDATE correctly blocked under the held transition: %v", err)
}

// TestConcurrency_RealGuardCellWriteBlockedByTransition drives the ACTUAL
// production cell-write lock function guardCellWrite (the exact code path the
// task_outcome + job_task use cases invoke before every leaf mutation) against a
// held transition. It proves the real writer protocol — parent FOR SHARE then
// job_phase FOR UPDATE — serializes against the transition's parent FOR UPDATE.
// This replaces the synthetic "cell write" proof the codex review flagged
// (§1 CONFIRMED: the old test never called a production mutation path).
func TestConcurrency_RealGuardCellWriteBlockedByTransition(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	a, _ := loadTwoSheets(t, db)
	ctx := context.Background()

	// Guard fails closed without an ambient transaction (a *sql.DB, not *sql.Tx).
	if err := guardCellWrite(ctx, db, "any-task", a.workspace); err == nil {
		t.Fatal("guardCellWrite must fail closed outside a transaction")
	}

	// Pick a live active job_task in sheet a.
	var jobTaskID string
	if err := db.QueryRowContext(ctx, `
		SELECT jt.id FROM job_task jt
		JOIN job_phase jp ON jp.id = jt.job_phase_id AND jp.active
		JOIN job j ON j.id = jp.job_id
		WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3 AND jt.active
		ORDER BY jt.id LIMIT 1`, a.templateID, a.phaseID, a.workspace).Scan(&jobTaskID); err != nil {
		t.Skipf("no active job_task in sheet a: %v", err)
	}

	// tx1 holds the transition's parent + sheet locks.
	tx1, _ := db.BeginTx(ctx, nil)
	defer tx1.Rollback()
	if err := lockSheet(ctx, tx1, a, ""); err != nil {
		t.Fatalf("tx1 lock sheet: %v", err)
	}

	// tx2 runs the REAL production guard — its parent FOR SHARE must block on the
	// transition's parent FOR UPDATE and time out.
	tx2, _ := db.BeginTx(ctx, nil)
	defer tx2.Rollback()
	if _, err := tx2.ExecContext(ctx, "SET LOCAL lock_timeout = '800ms'"); err != nil {
		t.Fatalf("set lock_timeout: %v", err)
	}
	start := time.Now()
	gerr := guardCellWrite(ctx, tx2, jobTaskID, a.workspace)
	if gerr == nil {
		t.Fatal("guardCellWrite acquired the cell lock while a transition held the sheet — real writer protocol NOT serialized")
	}
	t.Logf("REAL-GUARD: production guardCellWrite correctly blocked under the held transition after %s: %v",
		time.Since(start).Round(time.Millisecond), gerr)
}
