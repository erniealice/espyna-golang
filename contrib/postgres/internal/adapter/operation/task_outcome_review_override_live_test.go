//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Review-override live tests, A2 slice (plan 20261106, design-v2): fill records
// the ACTING staff with a strict audit; main-teacher (primary class edge) override
// of update. TEST_DATABASE_URL-gated; every write is inside a transaction that
// ends in ROLLBACK and the touched rows are compared afterwards.

// blankFixtureCell soft-deactivates an existing outcome (rolled back) so its cell
// is blank, and returns what a create-if-absent for that cell needs.
func blankFixtureCell(ctx context.Context, repo *PostgresTaskOutcomeRepository, outcomeID string) (*pb.TaskOutcome, error) {
	var jt, cv string
	if err := repo.executor(ctx).QueryRowContext(ctx,
		`SELECT job_task_id, criteria_version_id FROM `+entityid.TaskOutcome+` WHERE id = $1`, outcomeID).Scan(&jt, &cv); err != nil {
		return nil, err
	}
	if _, err := repo.executor(ctx).ExecContext(ctx,
		`UPDATE `+entityid.TaskOutcome+` SET active = false WHERE id = $1`, outcomeID); err != nil {
		return nil, err
	}
	v := 91.0
	return &pb.TaskOutcome{JobTaskId: jt, CriteriaVersionId: cv, NumericValue: &v}, nil
}

func fillAuditRow(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (perm, reason string, recordedBy sql.NullString, n int, err error) {
	rows, err := repo.executor(ctx).QueryContext(ctx,
		`SELECT COALESCE(e.permission_code,''), COALESCE(e.reason,''), f.new_value
		 FROM `+entityid.AuditEntry+` e
		 LEFT JOIN `+entityid.AuditFieldChange+` f ON f.audit_entry_id = e.id AND f.field_name = 'recorded_by'
		 WHERE e.entity_type = $1 AND e.entity_id = $2 ORDER BY e.occurred_at, e.id`, entityid.TaskOutcome, id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		n++
		if err = rows.Scan(&perm, &reason, &recordedBy); err != nil {
			return
		}
	}
	err = rows.Err()
	return
}

func TestReviewOverrideLive_FillStampsActor(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Fill: true}, false)
		probe, err := repo.probeCellOwnership(ctx, repo.executor(ctx).(*sql.Tx), fx.inTaskID, fx.ws, fx.reviewer, reviewedit.Decision{Fill: true})
		if err != nil || !probe.found || !probe.inScope {
			return fmt.Errorf("fixture must be in scope: %+v err=%v", probe, err)
		}
		if probe.own {
			t.Skip("fixture reviewer owns the cell by class edge/assignment; no override to prove")
		}
		data, err := blankFixtureCell(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		data.RecordedBy = "ignored-by-fill"
		rb := "ignored-reviewer"
		data.ReviewedBy = &rb
		out, err := repo.CreateTaskOutcomeIfAbsent(ctx, data)
		if err != nil {
			return fmt.Errorf("in-scope fill must succeed: %w", err)
		}
		st, err := readOutcomeState(ctx, repo, out.GetId())
		if err != nil {
			return err
		}
		if st.recordedBy.String != fx.reviewer {
			return fmt.Errorf("recorded_by = %q, want the acting reviewer", st.recordedBy.String)
		}
		if st.reviewedBy.Valid || st.reviewedDate.Valid {
			return fmt.Errorf("reviewed_by/date must be NULL on a fill")
		}
		perm, reason, rec, n, err := fillAuditRow(ctx, repo, out.GetId())
		if err != nil {
			return err
		}
		if n != 1 || perm != reviewedit.CreateCode || reason != "review_override_fill" || rec.String != fx.reviewer {
			return fmt.Errorf("audit rows=%d perm=%q reason=%q recorded_by=%q", n, perm, reason, rec.String)
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

func TestReviewOverrideLive_FillRefusals(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)

	t.Run("out_of_scope", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			d := reviewedit.Decision{Fill: true}
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, d, false)
			probe, err := repo.probeCellOwnership(ctx, repo.executor(ctx).(*sql.Tx), fx.outTaskID, fx.ws, fx.reviewer, d)
			if err != nil {
				return err
			}
			if probe.own {
				t.Skip("reviewer owns the out-of-scope cell by class edge; ordinary create path")
			}
			data, err := blankFixtureCell(ctx, repo, fx.outID)
			if err != nil {
				return err
			}
			if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, data); !errors.Is(err, portsdomain.ErrTaskOutcomeNotEditable) {
				return fmt.Errorf("out-of-scope fill must be NotEditable, got %v", err)
			}
			return errReviewEditRollback
		})
	})

	t.Run("locked_phase", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Fill: true}, false)
			if _, err := repo.executor(ctx).ExecContext(ctx,
				`UPDATE `+entityid.JobPhase+` SET approval_status = 'PHASE_APPROVAL_STATUS_FOR_REVIEW'
				 WHERE id = (SELECT job_phase_id FROM `+entityid.JobTask+` WHERE id = $1)`, fx.inTaskID); err != nil {
				return err
			}
			if err := repo.GuardCellWrite(ctx, fx.inTaskID); err == nil {
				return fmt.Errorf("a locked phase must be refused by the cell-write guard that precedes every fill")
			}
			return errReviewEditRollback
		})
	})

	t.Run("conflict", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Fill: true}, false)
			var jt, cv string
			if err := repo.executor(ctx).QueryRowContext(ctx,
				`SELECT job_task_id, criteria_version_id FROM `+entityid.TaskOutcome+` WHERE id = $1`, fx.inID).Scan(&jt, &cv); err != nil {
				return err
			}
			v := 1.0
			if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, &pb.TaskOutcome{JobTaskId: jt, CriteriaVersionId: cv, NumericValue: &v}); !errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
				return fmt.Errorf("an already recorded cell must conflict, got %v", err)
			}
			return errReviewEditRollback
		})
	})

	t.Run("plain_create_no_override", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			d := reviewedit.Decision{Fill: true}
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, d, false)
			probe, err := repo.probeCellOwnership(ctx, repo.executor(ctx).(*sql.Tx), fx.inTaskID, fx.ws, fx.reviewer, d)
			if err != nil {
				return err
			}
			if probe.own {
				t.Skip("reviewer owns the cell")
			}
			data, err := blankFixtureCell(ctx, repo, fx.inID)
			if err != nil {
				return err
			}
			if _, err := repo.CreateTaskOutcome(ctx, &pb.CreateTaskOutcomeRequest{Data: data}); !errors.Is(err, portsdomain.ErrTaskOutcomeNotEditable) {
				return fmt.Errorf("plain create by a non-owner holder must be refused, got %v", err)
			}
			return errReviewEditRollback
		})
	})
}

func TestReviewOverrideLive_FillAuditStrict(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)
	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		d := reviewedit.Decision{Fill: true}
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, d, false)
		probe, err := repo.probeCellOwnership(ctx, repo.executor(ctx).(*sql.Tx), fx.inTaskID, fx.ws, fx.reviewer, d)
		if err != nil {
			return err
		}
		if probe.own {
			t.Skip("reviewer owns the cell")
		}
		data, err := blankFixtureCell(ctx, repo, fx.inID)
		if err != nil {
			return err
		}
		ex := repo.executor(ctx)
		if _, err := ex.ExecContext(ctx, `CREATE FUNCTION public.zz_review_edit_fail_audit() RETURNS trigger LANGUAGE plpgsql AS
			$$ BEGIN RAISE EXCEPTION 'review-override live test: forced audit failure'; END $$`); err != nil {
			t.Skipf("cannot install the audit failure trigger: %v", err)
		}
		if _, err := ex.ExecContext(ctx, `CREATE TRIGGER zz_review_edit_fail_audit BEFORE INSERT ON `+entityid.AuditEntry+`
			FOR EACH ROW WHEN (NEW.entity_type = '`+entityid.TaskOutcome+`') EXECUTE FUNCTION public.zz_review_edit_fail_audit()`); err != nil {
			t.Skipf("cannot install the audit failure trigger: %v", err)
		}
		if _, err := repo.CreateTaskOutcomeIfAbsent(ctx, data); err == nil {
			return fmt.Errorf("an override fill whose audit fails must return an error (the use case then rolls back)")
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue:\nbefore %s\nafter  %s", before, after)
	}
}

// No fill code: the locked create is exactly today's (recorded_by as sent, ordinary
// audit tag, even for a cell outside any review scope).
func TestReviewOverrideLive_NoCodeUnchanged(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{}, false)
		data, err := blankFixtureCell(ctx, repo, fx.outID)
		if err != nil {
			return err
		}
		data.RecordedBy = fx.reviewer
		out, err := repo.CreateTaskOutcomeIfAbsent(ctx, data)
		if err != nil {
			return fmt.Errorf("without the fill decision the create is the ordinary one: %w", err)
		}
		st, _ := readOutcomeState(ctx, repo, out.GetId())
		if st.recordedBy.String != fx.reviewer {
			return fmt.Errorf("recorded_by = %q", st.recordedBy.String)
		}
		perm, reason, rec, _, err := fillAuditRow(ctx, repo, out.GetId())
		if err != nil {
			return err
		}
		if perm != "task_outcome:create" || reason != "" || rec.Valid {
			return fmt.Errorf("ordinary audit tag changed: perm=%q reason=%q recorded_by=%v", perm, reason, rec)
		}
		return errReviewEditRollback
	})
}

// Update tag: a review edit's audit row carries update_in_review_scope / review_override_update.
func TestReviewOverrideLive_UpdateTagRows(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	for _, mode := range []string{"plain", "conditional"} {
		t.Run(mode, func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
				var err error
				if mode == "plain" {
					err = plainUpdate(ctx, repo, fx.inID, reviewNote("tag-"+mode))
				} else {
					err = conditionalUpdate(ctx, repo, fx.inID, reviewNote("tag-"+mode))
				}
				if err != nil {
					return err
				}
				perm, reason, _, n, err := fillAuditRow(ctx, repo, fx.inID)
				if err != nil {
					return err
				}
				_ = n
				if perm != reviewedit.PermissionCode || reason != "review_override_update" {
					return fmt.Errorf("update audit tag perm=%q reason=%q", perm, reason)
				}
				return errReviewEditRollback
			})
		})
	}
}

// ---- main-teacher (primary class edge) override ----

type classFixture struct {
	ws, primary                string
	otherID, otherTask, other  string // outcome in the primary's class recorded by someone else
	secondary                  string // staff with only a secondary edge on that class
	coPrimaryID, coPrimaryTask string // outcome recorded by another PRIMARY of the same class (may be empty)
}

func pickClassFixture(t *testing.T, db *sql.DB) classFixture {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT staff_id, workspace_id FROM ` + entityid.SubscriptionGroupProductPlanStaff + `
		WHERE role = 'primary' AND active AND staff_id IS NOT NULL AND workspace_id IS NOT NULL`)
	if err != nil {
		t.Skipf("cannot list primary edges: %v", err)
	}
	type e struct{ staff, ws string }
	var edges []e
	for rows.Next() {
		var x e
		if err := rows.Scan(&x.staff, &x.ws); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		edges = append(edges, x)
	}
	rows.Close()
	for _, ed := range edges {
		q := `SELECT o.id, o.job_task_id, o.recorded_by
			FROM ` + entityid.TaskOutcome + ` o
			JOIN ` + entityid.JobTask + ` jt ON jt.id = o.job_task_id AND jt.active
			JOIN ` + entityid.JobPhase + ` jp ON jp.id = jt.job_phase_id AND jp.active
			     AND jp.approval_status = '` + reviewEditInProgress + `'
			JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
			WHERE o.active AND o.workspace_id = $2 AND COALESCE(o.recorded_by,'') <> '' AND o.recorded_by <> $1
			  AND COALESCE(o.reviewed_by,'') <> $1
			  AND ` + classPrimaryEdgeOwnedSQL(1, 2) + `
			LIMIT 1`
		fx := classFixture{ws: ed.ws, primary: ed.staff}
		if err := db.QueryRow(q, ed.staff, ed.ws).Scan(&fx.otherID, &fx.otherTask, &fx.other); err != nil {
			continue
		}
		// a staff with a class edge on the same cell but NOT a primary one
		srows, err := db.Query(`SELECT DISTINCT staff_id FROM `+entityid.SubscriptionGroupProductPlanStaff+`
			WHERE role = 'secondary' AND active AND workspace_id = $1`, ed.ws)
		if err == nil {
			var cands []string
			for srows.Next() {
				var c string
				_ = srows.Scan(&c)
				cands = append(cands, c)
			}
			srows.Close()
			for _, c := range cands {
				var edge, prim bool
				if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM `+entityid.JobTask+` jt
					JOIN `+entityid.JobPhase+` jp ON jp.id = jt.job_phase_id
					JOIN `+entityid.Job+` j ON j.id = jp.job_id
					WHERE jt.id = $1 AND `+classEdgeOwnedSQL(2, 3)+`), `+outcomeInClassPrimaryScopeSQL("$1", 2, 3),
					fx.otherTask, c, ed.ws).Scan(&edge, &prim); err == nil && edge && !prim && c != fx.other {
					fx.secondary = c
					break
				}
			}
		}
		return fx
	}
	t.Skip("no primary class-edge holder with an open outcome recorded by someone else")
	return classFixture{}
}

func classCtxDecision() reviewedit.Decision { return reviewedit.Decision{ClassOverride: true} }

func TestReviewOverrideLive_ClassPrimaryEdits(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickClassFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.otherID)

	for _, mode := range []string{"plain", "conditional"} {
		t.Run(mode, func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.primary, classCtxDecision(), false)
				note := reviewNote("class-" + mode)
				var err error
				if mode == "plain" {
					// the use case reads first: the class holder must be admitted by the scoped read
					if _, rerr := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: fx.otherID}}); rerr != nil {
						return fmt.Errorf("primary must read a class outcome: %w", rerr)
					}
					err = plainUpdate(ctx, repo, fx.otherID, note)
				} else {
					err = conditionalUpdate(ctx, repo, fx.otherID, note)
				}
				if err != nil {
					return fmt.Errorf("primary class override must succeed: %w", err)
				}
				st, err := readOutcomeState(ctx, repo, fx.otherID)
				if err != nil {
					return err
				}
				if st.note.String != *note || st.recordedBy.String != fx.other {
					return fmt.Errorf("note=%q recorded_by=%q (recorder must be unchanged: %q)", st.note.String, st.recordedBy.String, fx.other)
				}
				perm, reason, _, _, err := fillAuditRow(ctx, repo, fx.otherID)
				if err != nil {
					return err
				}
				if perm != reviewedit.ClassOverrideCode || reason != "class_override_update" {
					return fmt.Errorf("class audit tag perm=%q reason=%q", perm, reason)
				}
				return errReviewEditRollback
			})
		})
	}
	if after := outcomeImage(t, db, fx.otherID); after != before {
		t.Fatalf("rollback residue")
	}
}

func TestReviewOverrideLive_ClassSecondaryAndNoCodeRefused(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickClassFixture(t, db)
	repo, tm := reviewEditRepo(db)

	run := func(name, staff string, d reviewedit.Decision) {
		t.Run(name, func(t *testing.T) {
			if staff == "" {
				t.Skip("no fixture staff")
			}
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, staff, d, false)
				if err := conditionalUpdate(ctx, repo, fx.otherID, reviewNote(name)); err == nil {
					return fmt.Errorf("%s must be refused", name)
				}
				if _, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: fx.otherID}}); err == nil {
					return fmt.Errorf("%s must not read the outcome", name)
				}
				return errReviewEditRollback
			})
		})
	}
	run("secondary_with_code", fx.secondary, classCtxDecision())
	run("primary_without_code", fx.primary, reviewedit.Decision{})
	run("secondary_without_code", fx.secondary, reviewedit.Decision{})
}

// The review editor fills a blank; the class's primary teacher then changes it.
func TestReviewOverrideLive_PrimaryEditsReviewFilledCell(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		fd := reviewedit.Decision{Fill: true}
		rctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, fd, false)
		probe, err := repo.probeCellOwnership(rctx, repo.executor(rctx).(*sql.Tx), fx.inTaskID, fx.ws, fx.reviewer, fd)
		if err != nil || probe.own {
			t.Skipf("reviewer owns the cell or probe failed: %v", err)
		}
		// find a primary of this cell
		cands, err := repo.executor(rctx).QueryContext(rctx, `SELECT DISTINCT staff_id FROM `+entityid.SubscriptionGroupProductPlanStaff+`
			WHERE role = 'primary' AND active AND workspace_id = $1`, fx.ws)
		if err != nil {
			return err
		}
		var staffs []string
		for cands.Next() {
			var s string
			_ = cands.Scan(&s)
			staffs = append(staffs, s)
		}
		cands.Close()
		primary := ""
		for _, s := range staffs {
			var ok bool
			if err := repo.executor(rctx).QueryRowContext(rctx, `SELECT `+outcomeInClassPrimaryScopeSQL("$1", 2, 3), fx.inTaskID, s, fx.ws).Scan(&ok); err == nil && ok {
				primary = s
				break
			}
		}
		if primary == "" {
			t.Skip("no primary edge on the fixture cell")
		}
		data, err := blankFixtureCell(rctx, repo, fx.inID)
		if err != nil {
			return err
		}
		out, err := repo.CreateTaskOutcomeIfAbsent(rctx, data)
		if err != nil {
			return fmt.Errorf("review fill: %w", err)
		}
		tctx := reviewEditCtx(txCtx, fx.ws, primary, classCtxDecision(), false)
		if err := conditionalUpdate(tctx, repo, out.GetId(), reviewNote("teacher-after-fill")); err != nil {
			return fmt.Errorf("primary teacher must change the review-filled cell: %w", err)
		}
		st, _ := readOutcomeState(tctx, repo, out.GetId())
		if st.recordedBy.String != fx.reviewer {
			return fmt.Errorf("recorder must stay the filler, got %q", st.recordedBy.String)
		}
		return errReviewEditRollback
	})
}
