//go:build postgresql

package operation

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Every test here rolls back. The DSN comes from TEST_DATABASE_URL.

type condAuditFixture struct {
	repo *PostgresTaskOutcomeRepository
	tm   interfaces.TransactionManager
	db   *sql.DB
	tg   q26Target
}

func newCondAuditFixture(t *testing.T, db *sql.DB) *condAuditFixture {
	t.Helper()
	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)
	if repo.audit == nil {
		t.Fatalf("the repository must build its audit service")
	}
	return &condAuditFixture{repo: repo, tm: postgresCore.NewPostgreSQLTransactionManager(db), db: db, tg: pickQ26Target(t, db)}
}

// inTx runs fn in a transaction that always rolls back, with the row snapshot.
func (f *condAuditFixture) inTx(t *testing.T, fn func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error) {
	t.Helper()
	before := rowImage(t, f.db, f.tg.id)
	err := f.tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "cond-audit-test", WorkspaceID: f.tg.ws})
		wsID, tx, err := f.repo.conditionalTx(ctx, "test")
		if err != nil || wsID != f.tg.ws {
			return fmt.Errorf("ambient tx: %v", err)
		}
		read, err := f.repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: f.tg.id}})
		if err != nil || len(read.GetData()) == 0 {
			return fmt.Errorf("snapshot read: %v", err)
		}
		if err := fn(ctx, tx, read.GetData()[0]); err != nil {
			return err
		}
		return errQ26Rollback
	})
	if !errors.Is(err, errQ26Rollback) {
		t.Fatalf("transaction: %v", err)
	}
	if rowImage(t, f.db, f.tg.id) != before {
		t.Fatalf("the row changed after rollback")
	}
}

func auditRows(ctx context.Context, tx *sql.Tx, outcomeID string) (entries int, fields map[string][2]string, err error) {
	fields = map[string][2]string{}
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail.audit_entry
		WHERE entity_type = 'task_outcome' AND entity_id = $1 AND actor_id = 'cond-audit-test'
		  AND actor_type = 1 AND action = 2 AND domain = 'fayna'`, outcomeID).Scan(&entries); err != nil {
		return
	}
	rows, err := tx.QueryContext(ctx, `SELECT c.field_name, c.old_value, c.new_value
		FROM audit_trail.audit_field_change c JOIN audit_trail.audit_entry e ON e.id = c.audit_entry_id
		WHERE e.entity_type = 'task_outcome' AND e.entity_id = $1`, outcomeID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var oldV, newV sql.NullString
		if err = rows.Scan(&name, &oldV, &newV); err != nil {
			return
		}
		fields[name] = [2]string{oldV.String, newV.String}
	}
	return entries, fields, rows.Err()
}

func TestIntegrationConditionalAudit_RowsForPickAndRatingSave(t *testing.T) {
	f := newCondAuditFixture(t, openQ26DB(t))
	f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
		// A pick writes the note only.
		note := "picked wording for the audit test"
		out, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, DeterminationNote: &note}, snap)
		if err != nil {
			return fmt.Errorf("pick: %w", err)
		}
		n, fields, err := auditRows(ctx, tx, f.tg.id)
		if err != nil {
			return err
		}
		if n != 1 || len(fields) != 1 || fields["determination_note"] != [2]string{snap.GetDeterminationNote(), note} {
			return fmt.Errorf("pick audit: entries=%d fields=%d note-match=%v", n, len(fields), fields["determination_note"][1] == note)
		}

		// A rating save writes the value and the note.
		value := out.GetNumericValue() + 1
		note2 := "default wording for the new rating"
		out2, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, NumericValue: &value, DeterminationNote: &note2}, out)
		if err != nil {
			return fmt.Errorf("rating save: %w", err)
		}
		n, fields, err = auditRows(ctx, tx, f.tg.id)
		if err != nil {
			return err
		}
		if n != 2 || fields["numeric_value"][1] != fmt.Sprint(value) {
			return fmt.Errorf("rating audit: entries=%d numeric=%v", n, fields["numeric_value"])
		}

		// The same values again record nothing.
		if _, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, NumericValue: &value, DeterminationNote: &note2}, out2); err != nil {
			return fmt.Errorf("same-value save: %w", err)
		}
		if n, _, err = auditRows(ctx, tx, f.tg.id); err != nil || n != 2 {
			return fmt.Errorf("same-value save: entries=%d err=%v", n, err)
		}
		return nil
	})
}

// failingAudit fails as a real insert does: the statement errors inside the
// transaction, which aborts it unless a savepoint holds.
type failingAudit struct {
	infraports.AuditService
	tx    func() *sql.Tx
	calls int
}

func (a *failingAudit) LogEntry(ctx context.Context, _ *infraports.AuditLogRequest) error {
	a.calls++
	_, err := a.tx().ExecContext(ctx, `INSERT INTO audit_trail.no_such_table DEFAULT VALUES`)
	if err == nil {
		return errors.New("expected the statement to fail")
	}
	return err
}

func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// assertSaveSurvives writes through the repository and proves the transaction is
// healthy afterwards: the row reads back, and the error is logged at error level.
func assertSaveSurvives(ctx context.Context, f *condAuditFixture, tx *sql.Tx, snap *pb.TaskOutcome, buf *bytes.Buffer) error {
	value := snap.GetNumericValue() + 1
	note := "save survives a failed audit"
	out, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, NumericValue: &value, DeterminationNote: &note}, snap)
	if err != nil {
		return fmt.Errorf("the save must not fail: %w", err)
	}
	if out.GetDeterminationNote() != note || out.GetNumericValue() != value {
		return fmt.Errorf("returned row is wrong")
	}
	var gotNote string
	var gotValue float64
	if err := tx.QueryRowContext(ctx, `SELECT determination_note, numeric_value FROM task_outcome WHERE id = $1`, f.tg.id).Scan(&gotNote, &gotValue); err != nil {
		return fmt.Errorf("the transaction must stay healthy: %w", err)
	}
	if gotNote != note || gotValue != value {
		return fmt.Errorf("the update did not survive the audit rollback")
	}
	if !strings.Contains(buf.String(), "level=ERROR") {
		return fmt.Errorf("no error-level log: %q", buf.String())
	}
	return nil
}

func TestIntegrationConditionalAudit_FailureNeverFailsTheSave(t *testing.T) {
	t.Run("insert error aborts only the savepoint", func(t *testing.T) {
		f := newCondAuditFixture(t, openQ26DB(t))
		buf := captureSlog(t)
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			fail := &failingAudit{tx: func() *sql.Tx { return tx }}
			f.repo.audit = fail
			if err := assertSaveSurvives(ctx, f, tx, snap, buf); err != nil {
				return err
			}
			if fail.calls != 1 {
				return fmt.Errorf("audit calls = %d, want 1", fail.calls)
			}
			return nil
		})
	})

	// audit_entry is partitioned by month. With no partition for this month and no
	// default, the real adapter's insert fails with "no partition of relation".
	t.Run("missing month partition", func(t *testing.T) {
		f := newCondAuditFixture(t, openQ26DB(t))
		buf := captureSlog(t)
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			rows, err := tx.QueryContext(ctx, `SELECT c.oid::regclass::text FROM pg_inherits i
				JOIN pg_class c ON c.oid = i.inhrelid
				WHERE i.inhparent = 'audit_trail.audit_entry'::regclass
				  AND c.relname IN ('audit_entry_default', 'audit_entry_' || to_char(now(), 'YYYY_MM'))`)
			if err != nil {
				return err
			}
			var parts []string
			for rows.Next() {
				var p string
				if err := rows.Scan(&p); err != nil {
					rows.Close()
					return err
				}
				parts = append(parts, p)
			}
			rows.Close()
			if len(parts) == 0 {
				return fmt.Errorf("no partition found to detach")
			}
			for _, p := range parts {
				if _, err := tx.ExecContext(ctx, `ALTER TABLE audit_trail.audit_entry DETACH PARTITION `+p); err != nil {
					return fmt.Errorf("detach: %w", err)
				}
			}
			if err := assertSaveSurvives(ctx, f, tx, snap, buf); err != nil {
				return err
			}
			if !strings.Contains(buf.String(), "no partition of relation") {
				return fmt.Errorf("expected the missing-partition error, got %q", buf.String())
			}
			return nil
		})
	})
}

// recordingDriver logs each statement on a real pq connection, so the test can name
// exactly what the audit adds to a save.
type recordingDriver struct {
	mu    sync.Mutex
	stmts []string
	on    atomic.Bool
}

func (d *recordingDriver) note(q string) {
	if d.on.Load() {
		d.mu.Lock()
		d.stmts = append(d.stmts, strings.Join(strings.Fields(q), " "))
		d.mu.Unlock()
	}
}

func (d *recordingDriver) Open(name string) (driver.Conn, error) {
	c, err := pq.Open(name)
	if err != nil {
		return nil, err
	}
	return &recordingConn{Conn: c, d: d}, nil
}

type recordingConn struct {
	driver.Conn
	d *recordingDriver
}

func (c *recordingConn) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	c.d.note(q)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, args)
}

func (c *recordingConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.d.note(q)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func (c *recordingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}

var registerRecordingDriver sync.Once
var theRecordingDriver = &recordingDriver{}

func TestIntegrationConditionalAudit_AddsOnlyTheAuditStatements(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	registerRecordingDriver.Do(func() { sql.Register("pq-recording-cond-audit", theRecordingDriver) })
	db, err := sql.Open("pq-recording-cond-audit", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	d := theRecordingDriver

	f := newCondAuditFixture(t, db)
	withAudit := f.repo.audit
	measure := func(audit infraports.AuditService) []string {
		t.Helper()
		f.repo.audit = audit
		var got []string
		f.inTx(t, func(ctx context.Context, _ *sql.Tx, snap *pb.TaskOutcome) error {
			note := "statement count"
			d.mu.Lock()
			d.stmts = nil
			d.mu.Unlock()
			d.on.Store(true)
			_, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, DeterminationNote: &note}, snap)
			d.on.Store(false)
			d.mu.Lock()
			got = append(got, d.stmts...)
			d.mu.Unlock()
			return err
		})
		return got
	}

	plain := measure(nil)
	audited := measure(withAudit)
	if len(plain) != 1 || !strings.HasPrefix(plain[0], "UPDATE task_outcome") {
		t.Fatalf("without audit: %d statements: %v", len(plain), plain)
	}
	if len(audited) != 8 || audited[0] != plain[0] {
		t.Fatalf("with audit: %d statements", len(audited))
	}
	for i, prefix := range []string{
		"SAVEPOINT ",
		"SELECT current_setting('lock_timeout')",
		"SELECT set_config('lock_timeout'",
		"INSERT INTO audit_trail.audit_entry",
		"INSERT INTO audit_trail.audit_field_change",
		"SELECT set_config('lock_timeout'",
		"RELEASE SAVEPOINT ",
	} {
		if !strings.HasPrefix(audited[i+1], prefix) {
			t.Errorf("statement %d = %.60q, want prefix %q", i+1, audited[i+1], prefix)
		}
	}
}

// entriesOf counts this test's audit rows of one action for an outcome.
func entriesOf(ctx context.Context, tx *sql.Tx, id string, action int) (n int, err error) {
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail.audit_entry
		WHERE entity_type = 'task_outcome' AND entity_id = $1 AND actor_id = 'cond-audit-test' AND action = $2`,
		id, action).Scan(&n)
	return n, err
}

func lockTimeoutOf(ctx context.Context, tx *sql.Tx) (v string, err error) {
	err = tx.QueryRowContext(ctx, `SHOW lock_timeout`).Scan(&v)
	return v, err
}

// healthy proves the transaction still runs statements and the row reads as wanted.
func healthy(ctx context.Context, tx *sql.Tx, id string, active bool, note *string) error {
	var gotActive bool
	var gotNote sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT active, determination_note FROM task_outcome WHERE id = $1`, id).Scan(&gotActive, &gotNote); err != nil {
		return fmt.Errorf("the transaction must stay healthy: %w", err)
	}
	if gotActive != active {
		return fmt.Errorf("active = %v, want %v", gotActive, active)
	}
	if note != nil && gotNote.String != *note {
		return fmt.Errorf("the note did not stand")
	}
	return nil
}

// typedNote saves through the generic partial update, as saveDeterminationNote does.
func typedNote(ctx context.Context, f *condAuditFixture, note string) error {
	_, err := f.repo.UpdateTaskOutcome(ctx, &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: f.tg.id, DeterminationNote: &note}})
	return err
}

// clearCell deletes through the repository, as deleteCell does.
func clearCell(ctx context.Context, f *condAuditFixture) error {
	_, err := f.repo.DeleteTaskOutcome(ctx, &pb.DeleteTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: f.tg.id}})
	return err
}

// createCell clears the row, then creates the cell anew through the locked create.
func createCell(ctx context.Context, f *condAuditFixture, tx *sql.Tx, snap *pb.TaskOutcome) (*pb.TaskOutcome, error) {
	if _, err := tx.ExecContext(ctx, `UPDATE task_outcome SET active = false WHERE id = $1`, f.tg.id); err != nil {
		return nil, err
	}
	value := snap.GetNumericValue()
	note := "created cell"
	active := true
	return f.repo.CreateTaskOutcomeIfAbsent(ctx, &pb.TaskOutcome{
		JobTaskId:         f.tg.jobTaskID,
		CriteriaVersionId: f.tg.criteriaID,
		CriteriaType:      snap.GetCriteriaType(),
		NumericValue:      &value,
		DeterminationNote: &note,
		RecordedBy:        snap.GetRecordedBy(),
		Active:            active,
	})
}

func TestIntegrationGradeWriteAudit_RowAppears(t *testing.T) {
	t.Run("typed note", func(t *testing.T) {
		f := newCondAuditFixture(t, openQ26DB(t))
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			note := "a typed note for the audit test"
			if err := typedNote(ctx, f, note); err != nil {
				return err
			}
			n, fields, err := auditRows(ctx, tx, f.tg.id)
			if err != nil {
				return err
			}
			if n != 1 || len(fields) != 1 || fields["determination_note"][1] != note {
				return fmt.Errorf("entries=%d fields=%d", n, len(fields))
			}
			if err := typedNote(ctx, f, note); err != nil {
				return err
			}
			if n, _, err = auditRows(ctx, tx, f.tg.id); err != nil || n != 1 {
				return fmt.Errorf("the same note again: entries=%d err=%v", n, err)
			}
			return nil
		})
	})
	t.Run("cleared cell", func(t *testing.T) {
		f := newCondAuditFixture(t, openQ26DB(t))
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			if err := clearCell(ctx, f); err != nil {
				return err
			}
			n, err := entriesOf(ctx, tx, f.tg.id, 3)
			if err != nil || n != 1 {
				return fmt.Errorf("delete entries=%d err=%v", n, err)
			}
			_, fields, err := auditRows(ctx, tx, f.tg.id)
			if err != nil {
				return err
			}
			if fields["numeric_value"][0] == "" || fields["numeric_value"][1] != "" {
				return fmt.Errorf("the old value was not kept")
			}
			return healthy(ctx, tx, f.tg.id, false, nil)
		})
	})
	t.Run("created cell", func(t *testing.T) {
		f := newCondAuditFixture(t, openQ26DB(t))
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			created, err := createCell(ctx, f, tx, snap)
			if err != nil {
				return err
			}
			n, err := entriesOf(ctx, tx, created.GetId(), 1)
			if err != nil || n != 1 {
				return fmt.Errorf("insert entries=%d err=%v", n, err)
			}
			_, fields, err := auditRows(ctx, tx, created.GetId())
			if err != nil {
				return err
			}
			if fields["determination_note"][1] != "created cell" || fields["numeric_value"][1] == "" {
				return fmt.Errorf("the new values were not kept")
			}
			return nil
		})
	})
}

// A real insert error, so the transaction would abort without the savepoint.
func TestIntegrationGradeWriteAudit_SaveSurvivesAFailure(t *testing.T) {
	survive := func(name string, run func(ctx context.Context, f *condAuditFixture, tx *sql.Tx, snap *pb.TaskOutcome) error) {
		t.Run(name, func(t *testing.T) {
			f := newCondAuditFixture(t, openQ26DB(t))
			buf := captureSlog(t)
			f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
				fail := &failingAudit{tx: func() *sql.Tx { return tx }}
				f.repo.audit = fail
				if err := run(ctx, f, tx, snap); err != nil {
					return err
				}
				if fail.calls != 1 {
					return fmt.Errorf("audit calls = %d", fail.calls)
				}
				if !strings.Contains(buf.String(), "level=ERROR") {
					return fmt.Errorf("no error-level log")
				}
				return nil
			})
		})
	}
	survive("typed note", func(ctx context.Context, f *condAuditFixture, tx *sql.Tx, _ *pb.TaskOutcome) error {
		note := "typed note survives"
		if err := typedNote(ctx, f, note); err != nil {
			return fmt.Errorf("the save must not fail: %w", err)
		}
		return healthy(ctx, tx, f.tg.id, true, &note)
	})
	survive("cleared cell", func(ctx context.Context, f *condAuditFixture, tx *sql.Tx, _ *pb.TaskOutcome) error {
		if err := clearCell(ctx, f); err != nil {
			return fmt.Errorf("the clear must not fail: %w", err)
		}
		return healthy(ctx, tx, f.tg.id, false, nil)
	})
	survive("created cell", func(ctx context.Context, f *condAuditFixture, tx *sql.Tx, snap *pb.TaskOutcome) error {
		created, err := createCell(ctx, f, tx, snap)
		if err != nil {
			return fmt.Errorf("the create must not fail: %w", err)
		}
		note := "created cell"
		return healthy(ctx, tx, created.GetId(), true, &note)
	})
}

// The read before the write fails too: the save stands and writes no audit row.
func TestIntegrationGradeWriteAudit_FailedBeforeReadDoesNotFailTheSave(t *testing.T) {
	f := newCondAuditFixture(t, openQ26DB(t))
	buf := captureSlog(t)
	f.inTx(t, func(ctx context.Context, tx *sql.Tx, _ *pb.TaskOutcome) error {
		// Postgres refuses a NUL in a text parameter, so this read fails for real and
		// aborts the transaction unless the savepoint holds.
		if got := f.repo.readTaskOutcomeForAudit(ctx, "bad\x00id"); got != nil {
			return fmt.Errorf("a failed read returned a row")
		}
		if !strings.Contains(buf.String(), "level=ERROR") {
			return fmt.Errorf("no error-level log")
		}
		note := "still fine"
		if err := typedNote(ctx, f, note); err != nil {
			return err
		}
		return healthy(ctx, tx, f.tg.id, true, &note)
	})
}

// A row that is not there is not an audit failure: nothing at error level, and no
// claim that a save stood.
func TestIntegrationGradeWriteAudit_MissingRowIsQuiet(t *testing.T) {
	f := newCondAuditFixture(t, openQ26DB(t))
	buf := captureSlog(t)
	f.inTx(t, func(ctx context.Context, tx *sql.Tx, _ *pb.TaskOutcome) error {
		if got := f.repo.readTaskOutcomeForAudit(ctx, "not-a-row"); got != nil {
			return fmt.Errorf("read a row that does not exist")
		}
		logged := buf.String()
		if strings.Contains(logged, "level=ERROR") || strings.Contains(logged, "the save stands") {
			return fmt.Errorf("a missing row was logged as a failure")
		}
		if !strings.Contains(logged, "no row to read before the write") {
			return fmt.Errorf("the missing row was not logged")
		}
		return nil
	})
}

// Without an ambient transaction the save has committed before the audit runs, so no
// savepoint is taken and a failure stays a log line.
func TestIntegrationGradeWriteAudit_NoTransactionNeedsNoSavepoint(t *testing.T) {
	f := newCondAuditFixture(t, openQ26DB(t))
	buf := captureSlog(t)
	called := 0
	f.repo.isolateTaskOutcomeAudit(context.Background(), f.tg.id, false, func() error {
		called++
		return fmt.Errorf("refused")
	})
	if called != 1 || !strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("called=%d log=%q", called, buf.String())
	}
}

func TestIntegrationGradeWriteAudit_LockTimeoutDoesNotLeakToTheSave(t *testing.T) {
	check := func(t *testing.T, audit func(tx *sql.Tx) infraports.AuditService, wantDuring string) {
		f := newCondAuditFixture(t, openQ26DB(t))
		f.inTx(t, func(ctx context.Context, tx *sql.Tx, snap *pb.TaskOutcome) error {
			if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout = '7s'`); err != nil {
				return err
			}
			spy := &lockSpy{tx: tx, inner: audit(tx)}
			f.repo.audit = spy
			note := "lock timeout scope"
			if _, err := f.repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: f.tg.id, DeterminationNote: &note}, snap); err != nil {
				return err
			}
			if spy.during != wantDuring {
				return fmt.Errorf("during the audit lock_timeout = %q, want %q", spy.during, wantDuring)
			}
			after, err := lockTimeoutOf(ctx, tx)
			if err != nil || after != "7s" {
				return fmt.Errorf("after the audit lock_timeout = %q err=%v, want 7s", after, err)
			}
			return nil
		})
	}
	t.Run("audit succeeds", func(t *testing.T) {
		check(t, func(*sql.Tx) infraports.AuditService { return nil }, taskOutcomeAuditLockTimeout)
	})
	t.Run("audit fails", func(t *testing.T) {
		captureSlog(t)
		check(t, func(tx *sql.Tx) infraports.AuditService { return &failingAudit{tx: func() *sql.Tx { return tx }} }, taskOutcomeAuditLockTimeout)
	})
}

// lockSpy reads the setting at the moment the audit runs, then behaves as inner.
type lockSpy struct {
	infraports.AuditService
	tx     *sql.Tx
	inner  infraports.AuditService
	during string
}

func (s *lockSpy) LogEntry(ctx context.Context, req *infraports.AuditLogRequest) error {
	s.during, _ = lockTimeoutOf(ctx, s.tx)
	if s.inner != nil {
		return s.inner.LogEntry(ctx, req)
	}
	return nil
}

func TestTaskOutcomeAuditChanged_NoEmptyEntries(t *testing.T) {
	for name, tc := range map[string]struct {
		before, after map[string]any
		want          bool
	}{
		"nothing":             {map[string]any{}, map[string]any{}, false},
		"same":                {map[string]any{"a": "1"}, map[string]any{"a": "1"}, false},
		"absent equals empty": {map[string]any{}, map[string]any{"determination_note": ""}, false},
		"empty equals absent": {map[string]any{"determination_note": ""}, map[string]any{}, false},
		"changed":             {map[string]any{"a": "1"}, map[string]any{"a": "2"}, true},
		"added":               {map[string]any{}, map[string]any{"a": "1"}, true},
		"removed":             {map[string]any{"a": "1"}, map[string]any{}, true},
	} {
		if got := taskOutcomeAuditChanged(tc.before, tc.after); got != tc.want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}
