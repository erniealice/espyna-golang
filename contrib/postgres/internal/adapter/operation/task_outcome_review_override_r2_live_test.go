//go:build postgresql

package operation

import (
	"context"
	"errors"
	"fmt"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	touc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/task_outcome"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Review R2 live tests (plan 20261106-review-editor-override, integration):
// the plain delete stays recorder-only for every holder, the plain edit never
// widens for Clear or Fill alone, and the class override never clears. They run
// the REAL delete use cases on the postgres adapter, joined to an outer
// transaction that always rolls back (TEST_DATABASE_URL, override_check only).

// liveTx adapts the transaction manager to ports.Transactor; a nested call
// joins the outer (rolled-back) transaction.
type liveTx struct {
	tm *postgresCore.PostgreSQLTransactionManager
}

func (l liveTx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return l.tm.RunInTransaction(ctx, fn)
}
func (l liveTx) SupportsTransactions() bool               { return true }
func (l liveTx) IsTransactionActive(context.Context) bool { return false }

// codeAuthz grants exactly the listed codes on the fresh strict check.
type codeAuthz struct {
	ports.Authorizer
	codes map[string]bool
}

func (c codeAuthz) HasPermissionStrictFresh(_ context.Context, _ string, code string) (bool, error) {
	return c.codes[code], nil
}

func liveDeleteServices(tm *postgresCore.PostgreSQLTransactionManager, codes map[string]bool) touc.DeleteTaskOutcomeServices {
	tr := ports.NewNoOpTranslator()
	return touc.DeleteTaskOutcomeServices{
		Authorizer:       codeAuthz{codes: codes},
		Transactor:       liveTx{tm: tm},
		Translator:       tr,
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), tr),
	}
}

// staffSessionCtx is reviewEditCtx plus a user id, so the use cases' own
// reviewedit.Resolve runs; the pre-loaded decision is the strongest possible
// (every flag), which the plain delete must discard.
func staffSessionCtx(base context.Context, ws, staff string) context.Context {
	// No approval_scope:workspace marker: that marker already lifts the staff row
	// scope at b388d68f (principalscope.StaffRowScope), a pre-existing axis.
	ctx := reviewEditCtx(base, ws, staff, reviewedit.Decision{Allowed: true, Fill: true, Clear: true, ClassOverride: true, WorkspaceScope: true}, false)
	return contextutil.WithUserID(ctx, "review-edit-live-"+staff)
}

var (
	codesClearOnly  = map[string]bool{reviewedit.DeleteCode: true}
	codesUpdateOnly = map[string]bool{reviewedit.PermissionCode: true}
	codesFillOnly   = map[string]bool{reviewedit.CreateCode: true}
	codesClassOnly  = map[string]bool{reviewedit.ClassOverrideCode: true}
	codesAll        = map[string]bool{reviewedit.PermissionCode: true, reviewedit.CreateCode: true, reviewedit.DeleteCode: true, reviewedit.ClassOverrideCode: true}
)

func outcomeActive(ctx context.Context, repo *PostgresTaskOutcomeRepository, id string) (bool, error) {
	var a bool
	err := repo.executor(ctx).QueryRowContext(ctx, `SELECT active FROM task_outcome WHERE id = $1`, id).Scan(&a)
	return a, err
}

// Plain delete (/action/outcome/delete) is recorder-only for everyone: a
// Clear-only, update-only, fill-only or all-codes review holder, and a main
// teacher with the class override, are all refused on another recorder's grade
// in their scope; the recorder's own plain delete works (b388d68f behaviour).
func TestReviewOverrideR2Live_PlainDeleteRecorderOnly(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	refuse := func(name, ws, staff, id string, codes map[string]bool) {
		t.Run(name, func(t *testing.T) {
			if staff == "" || id == "" {
				t.Skip("no fixture")
			}
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := staffSessionCtx(txCtx, ws, staff)
				uc := touc.NewDeleteTaskOutcomeUseCase(touc.DeleteTaskOutcomeRepositories{TaskOutcome: repo}, liveDeleteServices(tm, codes))
				if _, err := uc.Execute(ctx, &pb.DeleteTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}}); err == nil {
					return fmt.Errorf("%s: plain delete of another recorder's grade must be refused", name)
				}
				if a, err := outcomeActive(txCtx, repo, id); err != nil || !a {
					return fmt.Errorf("%s: row must stay active (active=%v err=%v)", name, a, err)
				}
				return errReviewEditRollback
			})
		})
	}
	refuse("clear_only_holder", fx.ws, fx.reviewer, fx.inID, codesClearOnly)
	refuse("update_only_holder", fx.ws, fx.reviewer, fx.inID, codesUpdateOnly)
	refuse("fill_only_holder", fx.ws, fx.reviewer, fx.inID, codesFillOnly)
	refuse("all_codes_holder", fx.ws, fx.reviewer, fx.inID, codesAll)
	refuse("no_code_reviewer", fx.ws, fx.reviewer, fx.inID, nil)

	t.Run("recorder_own_plain_delete", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := staffSessionCtx(txCtx, fx.ws, fx.inRecorder)
			uc := touc.NewDeleteTaskOutcomeUseCase(touc.DeleteTaskOutcomeRepositories{TaskOutcome: repo}, liveDeleteServices(tm, nil))
			if _, err := uc.Execute(ctx, &pb.DeleteTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: fx.inID}}); err != nil {
				return fmt.Errorf("recorder's own plain delete must work: %w", err)
			}
			if a, err := outcomeActive(txCtx, repo, fx.inID); err != nil || a {
				return fmt.Errorf("own delete must soft-delete (active=%v err=%v)", a, err)
			}
			return errReviewEditRollback
		})
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue on %s", fx.inID)
	}
}

// Plain and conditional edits: Clear alone and Fill alone never widen the read
// or the write (revoking the update code turns editing off — design-v2 rule 5);
// update_in_review_scope still does. The update use cases read first, so the
// read refusal is the plain-edit refusal.
func TestReviewOverrideR2Live_EditNeverWidensForClearOrFill(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	for _, c := range []struct {
		name string
		d    reviewedit.Decision
	}{
		{"clear_only", reviewedit.Decision{Clear: true}},
		{"fill_only", reviewedit.Decision{Fill: true}},
		{"clear_and_fill", reviewedit.Decision{Clear: true, Fill: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, c.d, false)
				if _, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: fx.inID}}); err == nil {
					return fmt.Errorf("%s: read for edit must be refused", c.name)
				}
				if err := conditionalUpdate(ctx, repo, fx.inID, reviewNote("r2-"+c.name)); err == nil {
					return fmt.Errorf("%s: conditional edit must be refused", c.name)
				}
				return errReviewEditRollback
			})
		})
	}
	t.Run("control_update_holder_edits", func(t *testing.T) {
		inReviewEditTx(t, tm, func(txCtx context.Context) error {
			ctx := reviewEditCtx(txCtx, fx.ws, fx.reviewer, reviewedit.Decision{Allowed: true}, false)
			if err := conditionalUpdate(ctx, repo, fx.inID, reviewNote("r2-control")); err != nil {
				return fmt.Errorf("update holder edit must still work: %w", err)
			}
			return errReviewEditRollback
		})
	})
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue on %s", fx.inID)
	}
}

// Override clear (DeleteTaskOutcomeIfUnchanged use case): only Clear clears
// another recorder's grade. An update-only holder and a main teacher (class
// override, design-v2 rule 4) are refused as not editable; the row stays.
func TestReviewOverrideR2Live_ConditionalDeleteOnlyForClear(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, fx.inID)

	run := func(name, ws, staff, id string, codes map[string]bool, wantOK bool) {
		t.Run(name, func(t *testing.T) {
			if staff == "" || id == "" {
				t.Skip("no fixture")
			}
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				// the caller's snapshot: a workspace-wide (unscoped) staff read
				var snap *pb.TaskOutcome
				raw, err := repo.ReadTaskOutcome(reviewEditCtx(txCtx, ws, staff, reviewedit.Decision{}, true), &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}})
				if err != nil || len(raw.GetData()) == 0 {
					return fmt.Errorf("snapshot: %v", err)
				}
				snap = raw.GetData()[0]
				ctx := staffSessionCtx(txCtx, ws, staff)
				uc := touc.NewDeleteTaskOutcomeIfUnchangedUseCase(touc.DeleteTaskOutcomeRepositories{TaskOutcome: repo}, liveDeleteServices(tm, codes))
				_, conflict, err := uc.Execute(ctx, &pb.DeleteTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: id}}, snap)
				active, aerr := outcomeActive(txCtx, repo, id)
				if aerr != nil {
					return aerr
				}
				if wantOK {
					if err != nil || conflict || active {
						return fmt.Errorf("%s: clear must succeed (conflict=%v err=%v active=%v)", name, conflict, err, active)
					}
				} else if !errors.Is(err, portsdomain.ErrTaskOutcomeNotEditable) || conflict || !active {
					return fmt.Errorf("%s: want not editable and row intact, got conflict=%v err=%v active=%v", name, conflict, err, active)
				}
				return errReviewEditRollback
			})
		})
	}
	run("update_only_holder", fx.ws, fx.reviewer, fx.inID, codesUpdateOnly, false)
	run("fill_only_holder", fx.ws, fx.reviewer, fx.inID, codesFillOnly, false)
	run("no_code_reviewer", fx.ws, fx.reviewer, fx.inID, nil, false)
	run("clear_holder_out_of_scope", fx.ws, fx.reviewer, fx.outID, codesClearOnly, false)
	run("clear_holder_in_scope", fx.ws, fx.reviewer, fx.inID, codesClearOnly, true)
	if after := outcomeImage(t, db, fx.inID); after != before {
		t.Fatalf("rollback residue on %s", fx.inID)
	}
}

// reviewed_by alone never admits a conditional clear without the Clear code
// (review R2 minor): a staff who reviewed the outcome but did not record it.
func TestReviewOverrideR2Live_ReviewedByNoClear(t *testing.T) {
	db := openReviewEditDB(t)
	fx := pickReviewEditFixture(t, db)
	repo, tm := reviewEditRepo(db)
	id, ws, reviewer := fx.inID, fx.ws, fx.reviewer
	before := outcomeImage(t, db, id)
	inReviewEditTx(t, tm, func(txCtx context.Context) error {
		// Stage reviewed_by = the reviewer inside the rolled-back tx (the clone has
		// no reviewed-by-only open outcome).
		if _, err := repo.executor(txCtx).ExecContext(txCtx, `UPDATE task_outcome SET reviewed_by = $2 WHERE id = $1`, id, reviewer); err != nil {
			return err
		}
		ctx := reviewEditCtx(txCtx, ws, reviewer, reviewedit.Decision{}, false)
		snap, err := readSnapshot(ctx, repo, id) // reviewed_by still reads (b388d68f)
		if err != nil {
			return fmt.Errorf("reviewer must still read the outcome: %w", err)
		}
		if _, err := repo.DeleteTaskOutcomeIfUnchanged(ctx, id, snap); !errors.Is(err, portsdomain.ErrTaskOutcomeNotEditable) {
			return fmt.Errorf("reviewed_by without Clear must be not editable, got %v", err)
		}
		if a, err := outcomeActive(txCtx, repo, id); err != nil || !a {
			return fmt.Errorf("row must stay active (active=%v err=%v)", a, err)
		}
		return errReviewEditRollback
	})
	if after := outcomeImage(t, db, id); after != before {
		t.Fatalf("rollback residue on %s", id)
	}
}

// Main teacher (primary class edge + task_outcome:update_in_class_scope) never
// clears another recorder's grade (design-v2 rule 4): neither the plain delete
// nor the override clear; the class override edit itself still works.
func TestReviewOverrideR2Live_MainTeacherNeverClears(t *testing.T) {
	db := openReviewEditDB(t)
	cfx := pickClassFixture(t, db)
	repo, tm := reviewEditRepo(db)
	before := outcomeImage(t, db, cfx.otherID)
	req := &pb.DeleteTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: cfx.otherID}}

	for _, c := range []struct {
		name  string
		codes map[string]bool
	}{{"class_only", codesClassOnly}, {"class_and_update", map[string]bool{reviewedit.ClassOverrideCode: true, reviewedit.PermissionCode: true}}} {
		t.Run(c.name, func(t *testing.T) {
			inReviewEditTx(t, tm, func(txCtx context.Context) error {
				raw, err := repo.ReadTaskOutcome(reviewEditCtx(txCtx, cfx.ws, cfx.primary, reviewedit.Decision{}, true), &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: cfx.otherID}})
				if err != nil || len(raw.GetData()) == 0 {
					return fmt.Errorf("snapshot: %v", err)
				}
				ctx := staffSessionCtx(txCtx, cfx.ws, cfx.primary)
				svc := liveDeleteServices(tm, c.codes)
				if _, err := touc.NewDeleteTaskOutcomeUseCase(touc.DeleteTaskOutcomeRepositories{TaskOutcome: repo}, svc).Execute(ctx, req); err == nil {
					return fmt.Errorf("plain delete by the main teacher must be refused")
				}
				_, conflict, err := touc.NewDeleteTaskOutcomeIfUnchangedUseCase(touc.DeleteTaskOutcomeRepositories{TaskOutcome: repo}, svc).Execute(ctx, req, raw.GetData()[0])
				if !errors.Is(err, portsdomain.ErrTaskOutcomeNotEditable) || conflict {
					return fmt.Errorf("override clear by the main teacher: want not editable, got conflict=%v err=%v", conflict, err)
				}
				if a, err := outcomeActive(txCtx, repo, cfx.otherID); err != nil || !a {
					return fmt.Errorf("row must stay active (active=%v err=%v)", a, err)
				}
				// control: the class override edit still works for the same teacher
				if err := conditionalUpdate(reviewEditCtx(txCtx, cfx.ws, cfx.primary, reviewedit.Decision{ClassOverride: true}, false), repo, cfx.otherID, reviewNote("r2-class-control")); err != nil {
					return fmt.Errorf("class override edit must still work: %w", err)
				}
				return errReviewEditRollback
			})
		})
	}
	if after := outcomeImage(t, db, cfx.otherID); after != before {
		t.Fatalf("rollback residue on %s", cfx.otherID)
	}
}
