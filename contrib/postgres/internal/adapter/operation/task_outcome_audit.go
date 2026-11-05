//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	audittrailenums "github.com/erniealice/esqyma/pkg/schema/v1/domain/audit_trail/enums"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Grade writes record their own audit here. The task_outcome repository is built on
// operations that do not audit, so these are the only audit rows these writes get:
// the conditional update, the locked create, the partial update behind the typed
// note, and the delete behind a cleared cell.
//
// An audit failure never fails the save. A failed statement would abort the whole
// transaction, so the audit runs in a savepoint; a failure rolls back to it and is
// logged at error level. Other audited writes in this codebase, such as the job_phase
// approvals, instead fail the write when the audit fails.
const (
	taskOutcomeAuditSavepoint = "task_outcome_audit"

	// taskOutcomeAuditLockTimeout bounds how long an audit statement waits on a lock
	// while the save holds its grade row and phase locks. Inserts into audit_entry only
	// wait on partition maintenance, so waiting longer than this would just make
	// teachers queue behind it, and a skipped row is logged at error level.
	taskOutcomeAuditLockTimeout = "1s"
)

// taskOutcomeAuditWrite describes one grade write. Before is nil on a create, After
// is nil on a delete.
type taskOutcomeAuditWrite struct {
	WorkspaceID string
	Action      audittrailenums.AuditAction
	Permission  string
	Method      string
	ID          string
	Before      *pb.TaskOutcome
	After       *pb.TaskOutcome
	// Strict makes an audit failure fail the write: the error is returned and the
	// caller's transaction rolls back. Set for a review edit (see reviewEditStrict).
	Strict bool
	// Reason is audit_entry.reason (plan 20261106 L2): empty for ordinary writes;
	// review_override_fill / review_override_update / review_override_clear for
	// review overrides.
	Reason string
	// IncludeRecorder adds recorded_by to the logged fields; set by a review-override
	// fill so the audit names the staff the cell was recorded as.
	IncludeRecorder bool
}

// isolateTaskOutcomeAudit runs fn so that its failure cannot reach the save, and so
// that its lock timeout ends with it. With an ambient transaction that takes a
// savepoint. Without one the save has already committed by the time fn runs, so a
// failure cannot reach it, and fn runs on the pool with no lock timeout because
// SET LOCAL needs a transaction.
//
// strict=false keeps the best-effort behaviour above and always returns nil.
// strict=true returns the audit error so the caller fails the write and its
// transaction rolls back.
func (r *PostgresTaskOutcomeRepository) isolateTaskOutcomeAudit(ctx context.Context, id string, strict bool, fn func() error) error {
	tx, inTx := r.executor(ctx).(*sql.Tx)
	if !inTx || tx == nil {
		if err := fn(); err != nil {
			slog.Error("task_outcome audit failed; the save stands", "outcome_id", id, "err", err)
			if strict {
				return err
			}
		}
		return nil
	}
	if _, err := tx.ExecContext(ctx, `SAVEPOINT `+taskOutcomeAuditSavepoint); err != nil {
		slog.Error("task_outcome audit skipped: savepoint failed; the save stands", "outcome_id", id, "err", err)
		if strict {
			return fmt.Errorf("task_outcome audit: savepoint: %w", err)
		}
		return nil
	}

	err := func() error {
		var previous string
		if err := tx.QueryRowContext(ctx, `SELECT current_setting('lock_timeout')`).Scan(&previous); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('lock_timeout', $1, true)`, taskOutcomeAuditLockTimeout); err != nil {
			return err
		}
		if err := fn(); err != nil {
			return err
		}
		// RELEASE keeps a SET LOCAL for the rest of the transaction, so it is put back
		// by hand. ROLLBACK TO, on the failure path, undoes it by itself.
		_, err := tx.ExecContext(ctx, `SELECT set_config('lock_timeout', $1, true)`, previous)
		return err
	}()
	if err != nil {
		slog.Error("task_outcome audit failed; the save stands", "outcome_id", id, "err", err)
		if _, rbErr := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT `+taskOutcomeAuditSavepoint); rbErr != nil {
			slog.Error("task_outcome audit: rollback to savepoint failed", "outcome_id", id, "err", rbErr)
			if strict {
				return fmt.Errorf("task_outcome audit: %w (rollback to savepoint: %v)", err, rbErr)
			}
			return nil
		}
	}
	if _, relErr := tx.ExecContext(ctx, `RELEASE SAVEPOINT `+taskOutcomeAuditSavepoint); relErr != nil {
		slog.Error("task_outcome audit: release savepoint failed", "outcome_id", id, "err", relErr)
		if strict && err == nil {
			return fmt.Errorf("task_outcome audit: release savepoint: %w", relErr)
		}
	}
	if strict && err != nil {
		return fmt.Errorf("task_outcome audit: %w", err)
	}
	return nil
}

// reviewEditStrict reports whether a write is a review edit whose audit must not
// fail silently: the review-edit decision allows it and the acting staff is not the
// outcome's recorder. Fail closed: an empty acting staff counts as a review edit.
func reviewEditStrict(ctx context.Context, before *pb.TaskOutcome) bool {
	d, _ := reviewedit.DecisionFromContext(ctx)
	if !d.Allowed {
		if d.ClassOverride {
			// Main-teacher override: strict for any row by another staff. Fail closed
			// exactly like the review edit: an unreadable previous row, or an empty
			// acting staff, is strict (the caller then refuses the write).
			staffID, _ := principalscope.ActingStaff(ctx)
			return before == nil || staffID == "" || before.GetRecordedBy() != staffID
		}
		return false
	}
	staffID, _ := principalscope.ActingStaff(ctx)
	return before == nil || staffID == "" || before.GetRecordedBy() != staffID
}

// readTaskOutcomeForAudit reads the row before a write that does not return its old
// values. Nil means no audit row will be written, and that is logged.
func (r *PostgresTaskOutcomeRepository) readTaskOutcomeForAudit(ctx context.Context, id string) *pb.TaskOutcome {
	if r.audit == nil {
		return nil
	}
	wsID := ""
	if idn, ok := identity.FromContext(ctx); ok && idn != nil {
		wsID = idn.WorkspaceID
	}
	var before *pb.TaskOutcome
	_ = r.isolateTaskOutcomeAudit(ctx, id, false, func() error {
		row, err := scanTaskOutcomeSingleRow(r.executor(ctx).QueryRowContext(ctx,
			`SELECT `+taskOutcomeReturningCols+` FROM `+entityid.TaskOutcome+`
			 WHERE id = $1 AND (NULLIF($2, '') IS NULL OR workspace_id = $2)`, id, wsID))
		if errors.Is(err, sql.ErrNoRows) {
			// Nothing to compare with, and no failure: the write that follows finds no
			// row either, and reports that itself.
			slog.Info("task_outcome audit: no row to read before the write", "outcome_id", id)
			return nil
		}
		if err != nil {
			return err
		}
		before = row
		return nil
	})
	return before
}

// auditTaskOutcomeWrite records one grade write. A write that changed nothing the
// audit tracks records nothing.
//
// With w.Strict an audit failure (or a missing audit service) is returned and the
// caller must fail the write; otherwise the result is always nil.
func (r *PostgresTaskOutcomeRepository) auditTaskOutcomeWrite(ctx context.Context, w taskOutcomeAuditWrite) error {
	if r.audit == nil {
		if w.Strict {
			return fmt.Errorf("task_outcome audit: audit service unavailable for a review edit (fail closed)")
		}
		return nil
	}
	oldData, newData := taskOutcomeAuditFields(w.Before), taskOutcomeAuditFields(w.After)
	if w.IncludeRecorder && w.After != nil {
		newData["recorded_by"] = w.After.GetRecordedBy()
	}
	if !taskOutcomeAuditChanged(oldData, newData) {
		return nil
	}
	ac, _ := infraports.GetAuditContext(ctx)
	if idn, ok := identity.FromContext(ctx); ok && idn != nil && idn.UserID != "" {
		ac.ActorID = idn.UserID
		ac.ActorType = "user"
	}
	auditCtx := infraports.WithAuditContext(ctx, ac)
	return r.isolateTaskOutcomeAudit(ctx, w.ID, w.Strict, func() error {
		return infraports.DiffAndLog(auditCtx, r.audit, infraports.DiffAndLogRequest{
			WorkspaceID:    w.WorkspaceID,
			EntityType:     entityid.TaskOutcome,
			EntityID:       w.ID,
			Domain:         "fayna",
			Action:         int32(w.Action),
			PermissionCode: w.Permission,
			UseCase:        w.Method,
			Reason:         w.Reason,
			MethodName:     "PostgresTaskOutcomeRepository." + w.Method,
			OldData:        oldData,
			NewData:        newData,
		})
	})
}

// taskOutcomeAuditChanged compares the way DiffAndLog does, where an absent field and
// an empty one are the same text, so a write it would diff to nothing is never logged
// as an entry with no field changes.
func taskOutcomeAuditChanged(oldData, newData map[string]any) bool {
	text := func(m map[string]any, k string) string {
		s, _ := m[k].(string)
		return s
	}
	for k := range newData {
		if text(oldData, k) != text(newData, k) {
			return true
		}
	}
	for k := range oldData {
		if text(oldData, k) != text(newData, k) {
			return true
		}
	}
	return false
}

// taskOutcomeAuditFields is the grade columns these writes can change, as text.
func taskOutcomeAuditFields(o *pb.TaskOutcome) map[string]any {
	fields := map[string]any{}
	if o == nil {
		return fields
	}
	if ct := o.GetCriteriaType(); ct != enumspb.CriteriaType_CRITERIA_TYPE_UNSPECIFIED {
		fields["criteria_type"] = ct.String()
	}
	if o.NumericValue != nil {
		fields["numeric_value"] = strconv.FormatFloat(*o.NumericValue, 'g', -1, 64)
	}
	if o.TextValue != nil {
		fields["text_value"] = *o.TextValue
	}
	if o.CategoricalValue != nil {
		fields["categorical_value"] = *o.CategoricalValue
	}
	if o.PassFailValue != nil {
		fields["pass_fail_value"] = strconv.FormatBool(*o.PassFailValue)
	}
	if o.DeterminationNote != nil {
		fields["determination_note"] = *o.DeterminationNote
	}
	return fields
}
