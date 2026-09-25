//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/protobuf/proto"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Q26 conditional writes (PD 20260925-criterion-descriptors-by-program-year,
// schema-proposal §9.1) against a real database. TEST_DATABASE_URL-gated;
// run it against the CLONE (education2clone20260925a), never education2.
// Every mutation happens inside transactions that end in ROLLBACK, and the
// target rows are compared byte-for-byte before/after (zero residue).

func openQ26DB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL unusable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// q26Target picks one live active outcome with a numeric value in a
// workspace (read-only discovery).
type q26Target struct {
	id, ws, jobTaskID, criteriaID string
}

func pickQ26Target(t *testing.T, db *sql.DB) q26Target {
	t.Helper()
	var tg q26Target
	err := db.QueryRow(`
		SELECT o.id, o.workspace_id, o.job_task_id, o.criteria_version_id
		FROM task_outcome o
		JOIN job_task jt ON jt.id = o.job_task_id AND jt.workspace_id = o.workspace_id
		WHERE o.active AND o.numeric_value IS NOT NULL AND o.workspace_id IS NOT NULL
		  AND o.criteria_version_id IS NOT NULL
		ORDER BY o.id LIMIT 1`).Scan(&tg.id, &tg.ws, &tg.jobTaskID, &tg.criteriaID)
	if err != nil {
		t.Skipf("no live numeric task_outcome to exercise: %v", err)
	}
	return tg
}

func rowImage(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT row_to_json(o)::text FROM task_outcome o WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("row image %s: %v", id, err)
	}
	return s
}

func countActiveCell(t *testing.T, db *sql.DB, jobTaskID, criteriaID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM task_outcome WHERE job_task_id = $1 AND criteria_version_id = $2 AND active`, jobTaskID, criteriaID).Scan(&n); err != nil {
		t.Fatalf("count cell: %v", err)
	}
	return n
}

var errQ26Rollback = errors.New("q26test: intentional rollback")

func TestIntegrationQ26_UpdateTaskOutcomeIfUnchanged(t *testing.T) {
	db := openQ26DB(t)
	tg := pickQ26Target(t, db)
	before := rowImage(t, db, tg.id)

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)

	// Fail closed without an ambient transaction.
	noTxCtx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
	if _, err := repo.UpdateTaskOutcomeIfUnchanged(noTxCtx, &pb.TaskOutcome{Id: tg.id}, &pb.TaskOutcome{}); err == nil || !strings.Contains(err.Error(), "ambient transaction") {
		t.Fatalf("no-tx call must fail closed, got %v", err)
	}

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
		read, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: tg.id}})
		if err != nil || len(read.GetData()) == 0 {
			return fmt.Errorf("snapshot read: %v", err)
		}
		snap := read.GetData()[0]
		if snap.DateModified == nil {
			return fmt.Errorf("snapshot has no date_modified")
		}
		newVal := *snap.NumericValue + 1
		note := "q26 conditional note"

		// 1. Matching snapshot (incl. ms-truncated date_modified) → lands.
		out, err := repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: tg.id, NumericValue: &newVal, DeterminationNote: &note}, snap)
		if err != nil {
			return fmt.Errorf("matching snapshot must update: %w", err)
		}
		if out.GetNumericValue() != newVal || out.GetDeterminationNote() != note {
			return fmt.Errorf("RETURNING row = value %v note %q, want %v %q", out.GetNumericValue(), out.GetDeterminationNote(), newVal, note)
		}
		if out.GetJobTaskId() != tg.jobTaskID || out.GetCriteriaVersionId() != tg.criteriaID {
			return fmt.Errorf("membership anchors changed")
		}

		// 2. The now-stale snapshot → CONFLICT, row untouched.
		other := newVal + 1
		if _, err := repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: tg.id, NumericValue: &other}, snap); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("stale snapshot must CONFLICT, got %v", err)
		}
		var v float64
		var n sql.NullString
		if err := txQueryRow(ctx, repo, `SELECT numeric_value, determination_note FROM task_outcome WHERE id = $1`, tg.id).Scan(&v, &n); err != nil {
			return err
		}
		if v != newVal || n.String != note {
			return fmt.Errorf("conflict wrote: value %v note %q", v, n.String)
		}

		// 3. Fresh value but a note mismatch → CONFLICT (note is compared).
		fresh, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: tg.id}})
		if err != nil {
			return err
		}
		wrongNote := proto.Clone(fresh.GetData()[0]).(*pb.TaskOutcome)
		wn := "not the stored note"
		wrongNote.DeterminationNote = &wn
		if _, err := repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: tg.id, NumericValue: &other}, wrongNote); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("note mismatch must CONFLICT, got %v", err)
		}

		// 4. Foreign workspace → 0 rows → CONFLICT (never a cross-tenant write).
		foreign := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "q26", WorkspaceID: "q26-foreign-ws"})
		if _, err := repo.UpdateTaskOutcomeIfUnchanged(foreign, &pb.TaskOutcome{Id: tg.id, NumericValue: &other}, fresh.GetData()[0]); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("foreign workspace must not update, got %v", err)
		}
		return errQ26Rollback
	})
	if !errors.Is(err, errQ26Rollback) {
		t.Fatalf("expected the intentional rollback, got %v", err)
	}
	if after := rowImage(t, db, tg.id); after != before {
		t.Fatalf("rollback residue on %s:\nbefore %s\nafter  %s", tg.id, before, after)
	}
}

func TestIntegrationQ26_CreateTaskOutcomeIfAbsent(t *testing.T) {
	db := openQ26DB(t)
	tg := pickQ26Target(t, db)
	before := rowImage(t, db, tg.id)
	cellBefore := countActiveCell(t, db, tg.jobTaskID, tg.criteriaID)

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)
	newData := func() *pb.TaskOutcome {
		v := 4.0
		return &pb.TaskOutcome{JobTaskId: tg.jobTaskID, CriteriaVersionId: tg.criteriaID, NumericValue: &v, Active: true}
	}

	// A: occupied cell → CONFLICT; vacate (in-tx) → create; second create → CONFLICT.
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
		if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, newData()); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("occupied cell must CONFLICT, got %v", err)
		}
		if _, err := txExec(ctx, repo, `UPDATE task_outcome SET active = false WHERE job_task_id = $1 AND criteria_version_id = $2`, tg.jobTaskID, tg.criteriaID); err != nil {
			return err
		}
		created, err := repo.CreateTaskOutcomeIfAbsent(ctx, newData())
		if err != nil {
			return fmt.Errorf("vacant cell must create: %w", err)
		}
		if created.GetId() == "" {
			return fmt.Errorf("create returned no id")
		}
		var ws sql.NullString
		if err := txQueryRow(ctx, repo, `SELECT workspace_id FROM task_outcome WHERE id = $1`, created.GetId()).Scan(&ws); err != nil || ws.String != tg.ws {
			return fmt.Errorf("created row workspace = %q (err %v), want %q", ws.String, err, tg.ws)
		}
		if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, newData()); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("second create for the same cell must CONFLICT, got %v", err)
		}
		// Foreign workspace cannot lock/insert under this job_task.
		foreign := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "q26", WorkspaceID: "q26-foreign-ws"})
		if _, err := repo.CreateTaskOutcomeIfAbsent(foreign, newData()); err == nil || errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("foreign workspace must fail closed (not found), got %v", err)
		}
		return errQ26Rollback
	})
	if !errors.Is(err, errQ26Rollback) {
		t.Fatalf("A: expected the intentional rollback, got %v", err)
	}

	// B: two concurrent transactions — while A holds the job_task lock with an
	// uncommitted insert, B's create-if-absent must block on the lock (never
	// insert alongside). lock_timeout turns the block into an observable error.
	ctxA, cancelA := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelA()
	bDone := make(chan error, 1)
	errA := tm.RunInTransaction(ctxA, func(txA context.Context) error {
		ctx := identity.WithRequestIdentity(txA, &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
		if _, err := txExec(ctx, repo, `UPDATE task_outcome SET active = false WHERE job_task_id = $1 AND criteria_version_id = $2`, tg.jobTaskID, tg.criteriaID); err != nil {
			return err
		}
		if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, newData()); err != nil {
			return fmt.Errorf("A create: %w", err)
		}
		go func() {
			bDone <- tm.RunInTransaction(context.Background(), func(txB context.Context) error {
				bctx := identity.WithRequestIdentity(txB, &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
				if _, err := txExec(bctx, repo, `SET LOCAL lock_timeout = '750ms'`); err != nil {
					return err
				}
				_, err := repo.CreateTaskOutcomeIfAbsent(bctx, newData())
				if err == nil {
					return fmt.Errorf("B inserted while A held the job_task lock")
				}
				return err
			})
		}()
		select {
		case err := <-bDone:
			if err == nil || !strings.Contains(err.Error(), "lock") {
				return fmt.Errorf("B must block on A's job_task lock (lock_timeout), got %v", err)
			}
		case <-time.After(10 * time.Second):
			return fmt.Errorf("B neither blocked-out nor returned")
		}
		return errQ26Rollback
	})
	if !errors.Is(errA, errQ26Rollback) {
		t.Fatalf("B: expected the intentional rollback, got %v", errA)
	}

	if after := rowImage(t, db, tg.id); after != before {
		t.Fatalf("rollback residue on %s", tg.id)
	}
	if n := countActiveCell(t, db, tg.jobTaskID, tg.criteriaID); n != cellBefore {
		t.Fatalf("rollback residue: active outcomes for the cell %d, want %d", n, cellBefore)
	}
}

func txExec(ctx context.Context, r *PostgresTaskOutcomeRepository, q string, args ...any) (sql.Result, error) {
	return r.executor(ctx).ExecContext(ctx, q, args...)
}

func txQueryRow(ctx context.Context, r *PostgresTaskOutcomeRepository, q string, args ...any) *sql.Row {
	return r.executor(ctx).QueryRowContext(ctx, q, args...)
}

// fix2-backend (codex impl2 #10): nonnumeric edits that land inside the SAME
// millisecond as the snapshot must still CONFLICT — every typed value column
// (text/categorical/pass_fail) is compared, not only the ms-truncated
// date_modified. The same-ms condition is forced by restoring the row's exact
// stored date_modified after the first edit (a real concurrent writer in the
// same millisecond produces the identical ms image).
func TestIntegrationQ26_NonNumericSameMillisecondConflict(t *testing.T) {
	db := openQ26DB(t)
	tg := pickQ26Target(t, db)
	before := rowImage(t, db, tg.id)

	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)

	cases := []struct {
		name   string
		first  func() *pb.TaskOutcome
		second func() *pb.TaskOutcome
	}{
		{"text_value",
			func() *pb.TaskOutcome { s := "q26 text A"; return &pb.TaskOutcome{Id: tg.id, TextValue: &s} },
			func() *pb.TaskOutcome { s := "q26 text B"; return &pb.TaskOutcome{Id: tg.id, TextValue: &s} }},
		{"categorical_value",
			func() *pb.TaskOutcome { s := "q26-cat-A"; return &pb.TaskOutcome{Id: tg.id, CategoricalValue: &s} },
			func() *pb.TaskOutcome { s := "q26-cat-B"; return &pb.TaskOutcome{Id: tg.id, CategoricalValue: &s} }},
		{"pass_fail_value",
			func() *pb.TaskOutcome { b := true; return &pb.TaskOutcome{Id: tg.id, PassFailValue: &b} },
			func() *pb.TaskOutcome { b := false; return &pb.TaskOutcome{Id: tg.id, PassFailValue: &b} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
				ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "q26", WorkspaceID: tg.ws})
				// Normalize the typed columns to NULL so the first edit is a real change.
				if _, err := txExec(ctx, repo, `UPDATE task_outcome SET text_value = NULL, categorical_value = NULL, pass_fail_value = NULL WHERE id = $1`, tg.id); err != nil {
					return err
				}
				var stamp time.Time
				if err := txQueryRow(ctx, repo, `SELECT date_modified FROM task_outcome WHERE id = $1`, tg.id).Scan(&stamp); err != nil {
					return err
				}
				read, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: tg.id}})
				if err != nil || len(read.GetData()) == 0 {
					return fmt.Errorf("snapshot read: %v", err)
				}
				snap := read.GetData()[0] // both editors read this same snapshot

				if _, err := repo.UpdateTaskOutcomeIfUnchanged(ctx, tc.first(), snap); err != nil {
					return fmt.Errorf("first editor must land: %w", err)
				}
				// Same-millisecond: restore the exact pre-edit timestamp.
				if _, err := txExec(ctx, repo, `UPDATE task_outcome SET date_modified = $2 WHERE id = $1`, tg.id, stamp); err != nil {
					return err
				}
				if _, err := repo.UpdateTaskOutcomeIfUnchanged(ctx, tc.second(), snap); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
					return fmt.Errorf("second editor (stale %s, same ms) must CONFLICT, got %v", tc.name, err)
				}
				return errQ26Rollback
			})
			if !errors.Is(err, errQ26Rollback) {
				t.Fatalf("expected the intentional rollback, got %v", err)
			}
		})
	}
	if after := rowImage(t, db, tg.id); after != before {
		t.Fatalf("rollback residue on %s", tg.id)
	}
}
