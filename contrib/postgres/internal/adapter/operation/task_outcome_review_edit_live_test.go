//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Review-edit live tests (plan 20261105-reviewer-edit-grades; AC-3, AC-4, AC-5,
// AC-16). TEST_DATABASE_URL-gated (local clone only). Fixtures are discovered
// read-only; EVERY write happens inside a transaction that ends in ROLLBACK, and
// the touched rows are compared byte-for-byte afterwards. The identity context
// (kind-7 staff) and reviewedit.WithDecision are set directly, as the use cases
// would after resolving the fresh permission verdict.

var errReviewEditRollback = errors.New("review-edit live test: intentional rollback")

const reviewEditInProgress = "PHASE_APPROVAL_STATUS_IN_PROGRESS"

type reviewEditFixture struct {
	ws       string
	reviewer string // staff holding an active reviewer edge on the in-scope outcome's offering

	inID, inTaskID, inRecorder string // in review scope, recorded by someone else, reviewed_by empty, IN_PROGRESS phase

	outID, outTaskID, outRecorder string // same workspace, IN_PROGRESS, OUTSIDE the reviewer's scope, not recorded/reviewed by them
}

func openReviewEditDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping live review-edit test")
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

// pickReviewEditFixture walks the active reviewer edges (read-only) until one
// reviewer has BOTH an in-scope and an out-of-scope open outcome recorded by
// someone else. Skips when the database has none.
func pickReviewEditFixture(t *testing.T, db *sql.DB) reviewEditFixture {
	t.Helper()
	type edge struct{ staff, ws string }
	rows, err := db.Query(`SELECT DISTINCT staff_id, workspace_id FROM ` + entityid.ProductPlanStaff + `
		WHERE role = 'reviewer' AND active AND staff_id IS NOT NULL AND workspace_id IS NOT NULL`)
	if err != nil {
		t.Skipf("cannot list reviewer edges: %v", err)
	}
	var edges []edge
	for rows.Next() {
		var e edge
		if err := rows.Scan(&e.staff, &e.ws); err != nil {
			rows.Close()
			t.Fatalf("scan reviewer edge: %v", err)
		}
		edges = append(edges, e)
	}
	rows.Close()

	pick := func(e edge, inScope bool) (id, taskID, recorder string, ok bool) {
		scope := outcomeInReviewScopeSQL("o.job_task_id", 1, 2)
		cond := scope
		if !inScope {
			cond = "NOT " + scope + ` AND COALESCE(o.reviewed_by, '') <> $1`
		}
		q := `SELECT o.id, o.job_task_id, o.recorded_by
			FROM ` + entityid.TaskOutcome + ` o
			JOIN ` + entityid.JobTask + ` jt ON jt.id = o.job_task_id AND jt.active
			JOIN ` + entityid.JobPhase + ` jp ON jp.id = jt.job_phase_id AND jp.active
			     AND jp.approval_status = '` + reviewEditInProgress + `'
			WHERE o.active AND o.workspace_id = $2
			  AND COALESCE(o.recorded_by, '') <> '' AND o.recorded_by <> $1
			  AND COALESCE(o.reviewed_by, '') = ''
			  AND ` + cond + `
			LIMIT 1`
		err := db.QueryRow(q, e.staff, e.ws).Scan(&id, &taskID, &recorder)
		if err == sql.ErrNoRows {
			return "", "", "", false
		}
		if err != nil {
			t.Fatalf("pick outcome: %v", err)
		}
		return id, taskID, recorder, true
	}
	for _, e := range edges {
		fx := reviewEditFixture{ws: e.ws, reviewer: e.staff}
		var ok1, ok2 bool
		fx.inID, fx.inTaskID, fx.inRecorder, ok1 = pick(e, true)
		if !ok1 {
			continue
		}
		fx.outID, fx.outTaskID, fx.outRecorder, ok2 = pick(e, false)
		if !ok2 {
			continue
		}
		return fx
	}
	t.Skip("no reviewer with both an in-scope and an out-of-scope open outcome on this database")
	return reviewEditFixture{}
}

func reviewEditRepo(db *sql.DB) (*PostgresTaskOutcomeRepository, *postgresCore.PostgreSQLTransactionManager) {
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)
	return repo, tm.(*postgresCore.PostgreSQLTransactionManager)
}

// reviewEditCtx builds the request context: kind-7 staff identity plus the
// resolved review-edit decision. workspaceWide also marks the request with
// approval_scope:workspace (identity.WithWorkspaceRowScope).
func reviewEditCtx(base context.Context, ws, staff string, d reviewedit.Decision, workspaceWide bool) context.Context {
	ctx := identity.WithRequestIdentity(base, &identity.RequestIdentity{
		UserID:        "review-edit-live-" + staff,
		WorkspaceID:   ws,
		PrincipalType: principalscope.PrincipalTypeStaff,
		PrincipalID:   staff,
	})
	if workspaceWide {
		ctx = identity.WithWorkspaceRowScope(ctx)
	}
	return reviewedit.WithDecision(ctx, d)
}

func inReviewEditTx(t *testing.T, tm *postgresCore.PostgreSQLTransactionManager, fn func(txCtx context.Context) error) {
	t.Helper()
	err := tm.RunInTransaction(context.Background(), fn)
	if !errors.Is(err, errReviewEditRollback) {
		t.Fatalf("expected the intentional rollback, got %v", err)
	}
}

type outcomeState struct {
	note                  sql.NullString
	recordedBy            sql.NullString
	reviewedBy            sql.NullString
	reviewedDate          sql.NullTime
	active                bool
	exists                bool
	numericValue          sql.NullFloat64
	dateModifiedUnchanged bool
}

func readOutcomeState(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (outcomeState, error) {
	var s outcomeState
	err := repo.executor(ctx).QueryRowContext(ctx,
		`SELECT determination_note, recorded_by, reviewed_by, reviewed_date, active, numeric_value
		 FROM `+entityid.TaskOutcome+` WHERE id = $1`, id).
		Scan(&s.note, &s.recordedBy, &s.reviewedBy, &s.reviewedDate, &s.active, &s.numericValue)
	if err == sql.ErrNoRows {
		return s, nil
	}
	s.exists = err == nil
	return s, err
}

func reviewEditAuditRows(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (int, error) {
	var n int
	err := repo.executor(ctx).QueryRowContext(ctx,
		`SELECT count(*) FROM `+entityid.AuditEntry+` WHERE entity_type = $1 AND entity_id = $2`,
		entityid.TaskOutcome, id).Scan(&n)
	return n, err
}

func outcomeImage(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT row_to_json(o)::text FROM `+entityid.TaskOutcome+` o WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatalf("row image %s: %v", id, err)
	}
	return s
}

func reviewNote(tag string) *string {
	n := "review-edit-live " + tag
	return &n
}

// plainUpdate sends a review edit that ALSO tries to rewrite recorded_by /
// reviewed_by / reviewed_date: the adapter must strip them.
func plainUpdate(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string, note *string) error {
	hijackBy := "review-edit-live-hijack"
	hijackDate := time.Now().UnixMilli()
	_, err := repo.UpdateTaskOutcome(ctx, &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{
		Id: id, DeterminationNote: note,
		RecordedBy: hijackBy, ReviewedBy: &hijackBy, ReviewedDate: &hijackDate,
	}})
	return err
}

func conditionalUpdate(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string, note *string) error {
	read, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}})
	if err != nil {
		return fmt.Errorf("snapshot read: %w", err)
	}
	if len(read.GetData()) == 0 || read.GetData()[0].DateModified == nil {
		return fmt.Errorf("snapshot read returned no usable row")
	}
	_, err = repo.UpdateTaskOutcomeIfUnchanged(ctx, &pb.TaskOutcome{Id: id, DeterminationNote: note}, read.GetData()[0])
	return err
}

// AC-3 — in-scope reviewer edge updates another staff's outcome (plain and
// conditional): note changes, recorded_by unchanged, reviewed_by still NULL, an
// audit row is written.
func TestReviewEditLive_InScopeUpdateSucceeds(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	for _, mode := range []string{"plain", "conditional"} {
		t.Run(mode, func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
				pre, err := readOutcomeState(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				auditBefore, err := reviewEditAuditRows(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				note := reviewNote(mode)
				if mode == "plain" {
					err = plainUpdate(ctx, repo, fx.inID, note)
				} else {
					err = conditionalUpdate(ctx, repo, fx.inID, note)
				}
				if err != nil {
					return fmt.Errorf("in-scope %s review edit must succeed: %w", mode, err)
				}
				post, err := readOutcomeState(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				if post.note.String != *note {
					return fmt.Errorf("note = %q, want %q", post.note.String, *note)
				}
				if post.recordedBy.String != fx.inRecorder || post.recordedBy.String != pre.recordedBy.String {
					return fmt.Errorf("recorded_by changed: %q -> %q", pre.recordedBy.String, post.recordedBy.String)
				}
				if post.reviewedBy.Valid || post.reviewedDate.Valid != pre.reviewedDate.Valid {
					return fmt.Errorf("reviewed_by/date written by a review edit: by=%v date=%v", post.reviewedBy, post.reviewedDate)
				}
				auditAfter, err := reviewEditAuditRows(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				if auditAfter <= auditBefore {
					return fmt.Errorf("a review edit must write an audit_entry row (before %d, after %d)", auditBefore, auditAfter)
				}
				return errReviewEditRollback
			})
		})
	}
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue on %s:\nbefore %s\nafter  %s", fx.inID, before, after)
	}
}

// AC-3 — refusals: same actor on an offering they do not review; Allowed=false;
// no decision at all. Plain update and delete are gated by the use case's scoped
// ReadTaskOutcome (asserted here); the conditional UPDATE carries the staff clause.
func TestReviewEditLive_Refusals(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	type tc struct {
		name    string
		outcome string
		ctxFn   func(base context.Context) context.Context
		op      string // read | conditional
	}
	allowed := reviewedit.Decision{Allowed: true}
	cases := []tc{
		{"other_offering_plain", fx.outID, func(b context.Context) context.Context { return reviewEditCtx(b, fx.ws, fx.reviewer, allowed, false) }, "read"},
		{"other_offering_conditional", fx.outID, func(b context.Context) context.Context { return reviewEditCtx(b, fx.ws, fx.reviewer, allowed, false) }, "conditional"},
		{"not_allowed_plain", fx.inID, func(b context.Context) context.Context {
			return reviewEditCtx(b, fx.ws, fx.reviewer, reviewedit.Decision{}, false)
		}, "read"},
		{"not_allowed_conditional", fx.inID, func(b context.Context) context.Context {
			return reviewEditCtx(b, fx.ws, fx.reviewer, reviewedit.Decision{}, false)
		}, "conditional"},
		{"no_decision_plain", fx.inID, func(b context.Context) context.Context {
			return identity.WithRequestIdentity(b, &identity.RequestIdentity{
				UserID: "review-edit-live-" + fx.reviewer, WorkspaceID: fx.ws,
				PrincipalType: principalscope.PrincipalTypeStaff, PrincipalID: fx.reviewer,
			})
		}, "read"},
		// The delete use case puts no review-edit decision on ctx, so the scoped
		// read that gates it is not found for a non-owner.
		{"delete_gate_read_not_found", fx.inID, func(b context.Context) context.Context {
			return identity.WithRequestIdentity(b, &identity.RequestIdentity{
				UserID: "review-edit-live-" + fx.reviewer, WorkspaceID: fx.ws,
				PrincipalType: principalscope.PrincipalTypeStaff, PrincipalID: fx.reviewer,
			})
		}, "read"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := outcomeImage(t, db, c.outcome)
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := c.ctxFn(txCtx)
				var err error
				switch c.op {
				case "plain":
					err = plainUpdate(ctx, repo, c.outcome, reviewNote(c.name))
					if err == nil {
						return fmt.Errorf("plain update must be refused")
					}
				case "conditional":
					err = conditionalUpdate(ctx, repo, c.outcome, reviewNote(c.name))
					if err == nil {
						return fmt.Errorf("conditional update must be refused")
					}
				case "read":
					// The update/delete use cases gate on the scoped single read; the
					// adapter's generic Update/Delete carry no staff clause.
					if _, err = repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: c.outcome}}); err == nil {
						return fmt.Errorf("scoped read must return not found")
					}
				}
				// A refused statement can leave the tx aborted; the committed-state
				// comparison below is the authority, the in-tx read is best effort.
				if s, rerr := readOutcomeState(ctx, repo, c.outcome); rerr == nil {
					if s.note.String == *reviewNote(c.name) {
						return fmt.Errorf("refused edit left its note on the row")
					}
				}
				return errReviewEditRollback
			})
			if after := outcomeImage(t, db, c.outcome); after != before {
				t.Fatalf("refused case changed %s:\nbefore %s\nafter  %s", c.outcome, before, after)
			}
		})
	}
}

// AC-3 — a phase that is not IN_PROGRESS is refused by guardCellWrite (the lock
// protocol is unchanged for a reviewer). Control: IN_PROGRESS passes first.
func TestReviewEditLive_PhaseNotInProgressRefused(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
		if err := repo.GuardCellWrite(ctx, fx.inTaskID); err != nil {
			return fmt.Errorf("control: IN_PROGRESS phase must pass the guard: %w", err)
		}
		if _, err := repo.executor(ctx).ExecContext(ctx,
			`UPDATE `+entityid.JobPhase+` SET approval_status = 'PHASE_APPROVAL_STATUS_FOR_REVIEW'
			 WHERE id = (SELECT job_phase_id FROM `+entityid.JobTask+` WHERE id = $1)`, fx.inTaskID); err != nil {
			return fmt.Errorf("force FOR_REVIEW (rolled back): %w", err)
		}
		if err := repo.GuardCellWrite(ctx, fx.inTaskID); err == nil {
			return fmt.Errorf("a FOR_REVIEW phase must be refused by guardCellWrite even for an allowed reviewer")
		}
		return errReviewEditRollback
	})
}

// AC-5 — single read: in-scope reviewer sees the outcome, out-of-scope is not
// found; Allowed=false cannot see an other-recorder outcome either.
func TestReviewEditLive_ReadScope(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	read := func(ctx context.Context, id string) (found bool) {
		resp, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}})
		return err == nil && resp != nil && len(resp.GetData()) > 0 && resp.GetData()[0].GetId() == id
	}
	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		allowed := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
		if !read(allowed, fx.inID) {
			return fmt.Errorf("in-scope reviewer must read the outcome")
		}
		if read(allowed, fx.outID) {
			return fmt.Errorf("out-of-scope outcome must be not found for the reviewer")
		}
		denied := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{}, false)
		if read(denied, fx.inID) {
			return fmt.Errorf("without the decision the single read stays recorder/reviewer-scoped (today's behaviour)")
		}
		return errReviewEditRollback
	})
}

// AC-4 — strict audit: with the audit insert forced to fail (a trigger on
// audit_entry, created inside the rolled-back tx) a REVIEW edit errors and
// nothing persists, while a recorder's own edit still saves.
func TestReviewEditLive_AuditMandatory(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	forceAuditFailure := func(ctx context.Context) error {
		ex := repo.executor(ctx)
		if _, err := ex.ExecContext(ctx, `CREATE FUNCTION public.zz_review_edit_fail_audit() RETURNS trigger LANGUAGE plpgsql AS
			$$ BEGIN RAISE EXCEPTION 'review-edit live test: forced audit failure'; END $$`); err != nil {
			return err
		}
		_, err := ex.ExecContext(ctx, `CREATE TRIGGER zz_review_edit_fail_audit BEFORE INSERT ON `+entityid.AuditEntry+`
			FOR EACH ROW WHEN (NEW.entity_type = '`+entityid.TaskOutcome+`') EXECUTE FUNCTION public.zz_review_edit_fail_audit()`)
		return err
	}

	for _, mode := range []string{"plain", "conditional"} {
		t.Run("review_edit_fails_"+mode, func(t *testing.T) {
			before := outcomeImage(t, db, fx.inID)
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
				if err := forceAuditFailure(ctx); err != nil {
					t.Skipf("cannot install the audit failure trigger on this database: %v", err)
				}
				var err error
				if mode == "plain" {
					err = plainUpdate(ctx, repo, fx.inID, reviewNote("strict-"+mode))
				} else {
					err = conditionalUpdate(ctx, repo, fx.inID, reviewNote("strict-"+mode))
				}
				if err == nil {
					return fmt.Errorf("a review edit whose audit fails must return an error (%s)", mode)
				}
				if errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
					return fmt.Errorf("expected the audit error, got a conflict: %w", err)
				}
				// The use case returns this error, so its transaction rolls back.
				return errReviewEditRollback
			})
			if after := outcomeImage(t, db, fx.inID); after != before {
				t.Fatalf("nothing may persist after a failed review audit:\nbefore %s\nafter  %s", before, after)
			}
		})
	}

	// The recorder's own edit with the SAME audit failure still saves (best-effort audit).
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprintf("recorder_own_edit_still_saves_allowed_%v", allowed), func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.inRecorder, reviewedit.Decision{Allowed: allowed}, false)
				if err := forceAuditFailure(ctx); err != nil {
					t.Skipf("cannot install the audit failure trigger on this database: %v", err)
				}
				note := reviewNote("recorder-own")
				if err := plainOwnUpdate(ctx, repo, fx.inID, note); err != nil {
					return fmt.Errorf("recorder's own edit must still save when the audit fails: %w", err)
				}
				s, err := readOutcomeState(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				if s.note.String != *note {
					return fmt.Errorf("recorder's edit did not land: note %q", s.note.String)
				}
				return errReviewEditRollback
			})
		})
	}
}

// plainOwnUpdate is a recorder's ordinary edit (no recorded_by/reviewed_by in the payload).
func plainOwnUpdate(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string, note *string) error {
	_, err := repo.UpdateTaskOutcome(ctx, &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id, DeterminationNote: note}})
	return err
}

// AC-16 — workspace-wide staff (approval_scope:workspace row scope) WITH the
// decision (Allowed + WorkspaceScope) edits any open outcome in the workspace,
// including one outside any reviewer edge; recorded_by stays; a foreign
// workspace is refused.
func TestReviewEditLive_WorkspaceScope(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	dec := reviewedit.Decision{Allowed: true, WorkspaceScope: true}

	t.Run("edits_outside_reviewer_edge", func(t *testing.T) {
		before := outcomeImage(t, db, fx.outID)
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, dec, true)
			note := reviewNote("workspace")
			if err := plainUpdate(ctx, repo, fx.outID, note); err != nil {
				return fmt.Errorf("workspace-wide review edit must succeed: %w", err)
			}
			s, err := readOutcomeState(ctx, repo, fx.outID)
			if err != nil {
				return err
			}
			if s.note.String != *note {
				return fmt.Errorf("note = %q, want %q", s.note.String, *note)
			}
			if s.recordedBy.String != fx.outRecorder {
				return fmt.Errorf("recorded_by changed to %q", s.recordedBy.String)
			}
			if s.reviewedBy.Valid {
				return fmt.Errorf("reviewed_by written: %v", s.reviewedBy)
			}
			if err := conditionalUpdate(ctx, repo, fx.outID, reviewNote("workspace-cond")); err != nil {
				return fmt.Errorf("workspace-wide conditional review edit must succeed: %w", err)
			}
			return errReviewEditRollback
		})
		if after := outcomeImage(t, db, fx.outID); after != before {
			t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
		}
	})

	t.Run("foreign_workspace_refused", func(t *testing.T) {
		before := outcomeImage(t, db, fx.outID)
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, "review-edit-live-foreign-ws", fx.reviewer, dec, true)
			if err := plainUpdate(ctx, repo, fx.outID, reviewNote("foreign")); err == nil {
				return fmt.Errorf("plain review edit from a foreign workspace must be refused")
			}
			return errReviewEditRollback
		})
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, "review-edit-live-foreign-ws", fx.reviewer, dec, true)
			if err := conditionalUpdate(ctx, repo, fx.outID, reviewNote("foreign-cond")); err == nil {
				return fmt.Errorf("conditional review edit from a foreign workspace must be refused")
			}
			return errReviewEditRollback
		})
		if after := outcomeImage(t, db, fx.outID); after != before {
			t.Fatalf("foreign workspace changed the row:\nbefore %s\nafter  %s", before, after)
		}
	})
}
