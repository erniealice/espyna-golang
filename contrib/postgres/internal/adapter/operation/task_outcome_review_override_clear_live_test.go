//go:build postgresql

package operation

import (
	"context"
	"errors"
	"fmt"
	"testing"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Review-override clear live tests (plan 20261106-review-editor-override, E4;
// AC-O6, AC-O11). TEST_DATABASE_URL-gated (override_check clone only). Every
// write happens inside a transaction that ends in ROLLBACK; the touched row is
// compared byte-for-byte afterwards.

var clearDecision = reviewedit.Decision{Clear: true}

func readSnapshot(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (*pb.TaskOutcome, error) {
	read, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}})
	if err != nil {
		return nil, fmt.Errorf("snapshot read: %w", err)
	}
	if len(read.GetData()) == 0 {
		return nil, fmt.Errorf("snapshot read returned no row")
	}
	return read.GetData()[0], nil
}

type clearAuditRow struct {
	n          int
	permission string
	reason     string
	actor      string
}

func clearAudit(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (clearAuditRow, error) {
	var a clearAuditRow
	if err := repo.executor(ctx).QueryRowContext(ctx,
		`SELECT count(*) FROM audit_trail.audit_entry WHERE entity_type = 'task_outcome' AND entity_id = $1 AND action = 3`, id).Scan(&a.n); err != nil {
		return a, err
	}
	if a.n == 0 {
		return a, nil
	}
	err := repo.executor(ctx).QueryRowContext(ctx,
		`SELECT COALESCE(permission_code,''), COALESCE(reason,''), COALESCE(actor_id,'') FROM audit_trail.audit_entry
		 WHERE entity_type = 'task_outcome' AND entity_id = $1 AND action = 3 LIMIT 1`, id).
		Scan(&a.permission, &a.reason, &a.actor)
	return a, err
}

// AC-O6 / AC-O11 — a Clear holder in scope soft-deletes another staff's
// outcome: active=false, recorded_by untouched, strict audit row with the
// override permission and reason; the scoped read admits Clear.
func TestReviewOverrideLive_ClearSoftDeletes(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, clearDecision, false)
		snap, err := readSnapshot(ctx, repo, fx.inID) // inReviewEditScope admits Clear
		if err != nil {
			return err
		}
		if err := repo.GuardCellWrite(ctx, snap.GetJobTaskId()); err != nil {
			return fmt.Errorf("control: guard must pass on an open sheet: %w", err)
		}
		out, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, fx.inID, snap)
		if err != nil {
			return fmt.Errorf("in-scope clear must succeed: %w", err)
		}
		if out.GetId() != fx.inID {
			return fmt.Errorf("returned id %q", out.GetId())
		}
		post, err := readOutcomeState(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if post.active {
			return fmt.Errorf("outcome must be inactive after a clear")
		}
		if post.recordedBy.String != fx.inRecorder {
			return fmt.Errorf("recorded_by changed to %q", post.recordedBy.String)
		}
		a, err := clearAudit(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if a.n < 1 || a.permission != reviewedit.DeleteCode || a.reason != "review_override_clear" || a.actor != "review-edit-live-"+fx.reviewer {
			return fmt.Errorf("audit row = %+v, want DELETE by holder with %s / review_override_clear", a, reviewedit.DeleteCode)
		}
		// A second clear with the same snapshot finds the row inactive.
		if _, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, fx.inID, snap); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("second clear must conflict, got %v", err)
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// AC-O6 — a stale snapshot (changed value or timestamp) is a conflict and
// writes nothing.
func TestReviewOverrideLive_ClearSnapshotConflict(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, clearDecision, false)
		snap, err := readSnapshot(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		// A concurrent writer changes the note after the snapshot.
		if _, err := repo.executor(ctx).ExecContext(ctx,
			`UPDATE `+entityid.TaskOutcome+` SET determination_note = 'override-clear-live-concurrent', date_modified = now() WHERE id = $1`, fx.inID); err != nil {
			return err
		}
		if _, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, fx.inID, snap); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("stale snapshot must conflict, got %v", err)
		}
		s, err := readOutcomeState(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if !s.active {
			return fmt.Errorf("a conflicting clear must leave the row active")
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// AC-O6 — a Clear holder outside the review scope, a holder without the Clear
// flag (Allowed only / no decision), and a non-owner: refused, nothing written.
func TestReviewOverrideLive_ClearOutOfScopeRefused(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	cases := []struct {
		name    string
		outcome string
		d       reviewedit.Decision
	}{
		{"out_of_scope_clear", fx.outID, clearDecision},
		{"in_scope_no_clear_flag", fx.inID, reviewedit.Decision{Allowed: true, Fill: true}},
		{"in_scope_no_decision", fx.inID, reviewedit.Decision{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := outcomeImage(t, db, c.outcome)
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				// Snapshot taken with the full-scope ctx of the recorder-agnostic read:
				// build it straight from the row so the read scope cannot hide the case.
				snap := &pb.TaskOutcome{}
				ctx0 := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true, Clear: true, Fill: true}, true)
				s, err := readSnapshot(ctx0, repo, c.outcome)
				if err != nil {
					return err
				}
				snap = s
				ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, c.d, false)
				// Out of scope for a Clear holder is a scope refusal, not a stale snapshot.
				want := portsdomain.ErrTaskOutcomeConflict
				if c.name == "out_of_scope_clear" {
					want = portsdomain.ErrTaskOutcomeNotEditable
				}
				if _, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, c.outcome, snap); !errors.Is(err, want) {
					return fmt.Errorf("must be refused with %v, got %v", want, err)
				}
				return errReviewEditRollback
			})
			if after := outcomeImage(t, db, c.outcome); after != before {
				t.Fatalf("refused clear changed the row:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

// AC-O6 — a locked sheet is refused by GuardCellWrite, which the use case runs
// before the delete; the row is not touched.
func TestReviewOverrideLive_ClearLockedRefused(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, clearDecision, false)
		snap, err := readSnapshot(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if _, err := repo.executor(ctx).ExecContext(ctx,
			`UPDATE `+entityid.JobPhase+` SET approval_status = 'PHASE_APPROVAL_STATUS_FOR_REVIEW'
			 WHERE id = (SELECT job_phase_id FROM `+entityid.JobTask+` WHERE id = $1)`, snap.GetJobTaskId()); err != nil {
			return err
		}
		if err := repo.GuardCellWrite(ctx, snap.GetJobTaskId()); err == nil {
			return fmt.Errorf("a FOR_REVIEW phase must refuse the clear's guard")
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// AC-O11 — strict audit: with no audit service the override clear fails and
// the row stays active (the caller's transaction rolls back).
func TestReviewOverrideLive_ClearAuditStrict(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	noAudit := *repo
	noAudit.audit = nil
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, clearDecision, false)
		snap, err := readSnapshot(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if _, err := noAudit.DeleteTaskOutcomeIfUnchanged(ctx, fx.inID, snap); err == nil || errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("an override clear without an audit service must fail (not conflict), got %v", err)
		}
		return errReviewEditRollback // the use case's tx would roll back the UPDATE
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// AC-O6 — after a clear the cell is empty: the teacher (cell owner) can create
// the outcome again with create-if-absent.
func TestReviewOverrideLive_TeacherCanReenterAfterClear(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, clearDecision, false)
		snap, err := readSnapshot(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if _, err := repo.CreateTaskOutcomeIfAbsent(reviewEditCtx(txCtx, fx.ws, fx.inRecorder, reviewedit.Decision{}, false),
			&pb.TaskOutcome{JobTaskId: snap.GetJobTaskId(), CriteriaVersionId: snap.GetCriteriaVersionId(), NumericValue: snap.NumericValue}); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
			return fmt.Errorf("control: the cell is occupied before the clear, got %v", err)
		}
		if _, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, fx.inID, snap); err != nil {
			return fmt.Errorf("clear: %w", err)
		}
		tctx := reviewEditCtx(txCtx, fx.ws, fx.inRecorder, reviewedit.Decision{}, false)
		created, err := repo.CreateTaskOutcomeIfAbsent(tctx, &pb.TaskOutcome{
			JobTaskId: snap.GetJobTaskId(), CriteriaVersionId: snap.GetCriteriaVersionId(),
			CriteriaType: snap.GetCriteriaType(), NumericValue: snap.NumericValue, RecordedBy: fx.inRecorder,
		})
		if err != nil {
			return fmt.Errorf("teacher re-entry after a clear must succeed: %w", err)
		}
		if created.GetId() == fx.inID || !created.GetActive() {
			return fmt.Errorf("re-entry must create a new active row, got id=%s active=%v", created.GetId(), created.GetActive())
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// AC-O6 — no-code unchanged: with no Clear flag the conditional delete of an
// outcome the actor did not record never lands; the recorder's own conditional
// delete (no override) still works and keeps the plain audit tag.
func TestReviewOverrideLive_ClearNoCodeUnchanged(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		rctx := reviewEditCtx(txCtx, fx.ws, fx.inRecorder, reviewedit.Decision{}, false)
		snap, err := readSnapshot(rctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if _, err := repo.DeleteTaskOutcomeIfUnchanged(rctx, fx.inID, snap); err != nil {
			return fmt.Errorf("recorder's own conditional delete must work: %w", err)
		}
		a, err := clearAudit(rctx, repo, fx.inID)
		if err != nil {
			return err
		}
		if a.n < 1 || a.permission != "task_outcome:delete" || a.reason != "" {
			return fmt.Errorf("recorder delete audit = %+v, want plain task_outcome:delete without reason", a)
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}
