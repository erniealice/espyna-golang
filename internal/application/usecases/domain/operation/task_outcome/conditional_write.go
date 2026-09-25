package task_outcome

import (
	"context"
	"errors"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Q26 conditional grade-sheet writes (docs/plan/20260925-criterion-
// descriptors-by-program-year, schema-proposal.md §9.1, interfaces.md §2b).
//
// Both use cases return (response, conflict, error):
//   - conflict == true, err == nil → the cell changed (update) / already
//     exists (create) since the caller's snapshot; NOTHING was written. The
//     caller rejects the cell (fayna: `cell_changed_retry`).
//   - err != nil → an ordinary failure (authz, validation, guard, SQL).
//
// Provider contract (mirrors the cell-write guard's fail-closed rule): on a
// transactional provider the repository MUST implement BOTH the cell-write
// guard and portsdomain.TaskOutcomeConditionalWriter — otherwise the call
// fails closed. A non-transactional provider (mock) keeps a best-effort
// path: update compares the snapshot in memory then writes; create writes
// unconditionally (no concurrency exists there to protect against).

// UpdateTaskOutcomeIfUnchangedUseCase — conditional update of one outcome's
// value/type/note against the caller's snapshot (expected).
type UpdateTaskOutcomeIfUnchangedUseCase struct {
	base *UpdateTaskOutcomeUseCase
}

// NewUpdateTaskOutcomeIfUnchangedUseCase shares the plain update use case's
// repositories/services (same gatekeeper, transactor, translator).
func NewUpdateTaskOutcomeIfUnchangedUseCase(repositories UpdateTaskOutcomeRepositories, services UpdateTaskOutcomeServices) *UpdateTaskOutcomeIfUnchangedUseCase {
	return &UpdateTaskOutcomeIfUnchangedUseCase{base: NewUpdateTaskOutcomeUseCase(repositories, services)}
}

// Execute applies the conditional update. expected is the TaskOutcome the
// caller read earlier (numeric_value, determination_note, date_modified are
// compared; nil pointers mean "expected NULL").
func (uc *UpdateTaskOutcomeIfUnchangedUseCase) Execute(ctx context.Context, req *pb.UpdateTaskOutcomeRequest, expected *pb.TaskOutcome) (*pb.UpdateTaskOutcomeResponse, bool, error) {
	b := uc.base
	if err := b.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, false, err
	}
	if err := b.validateInput(ctx, req); err != nil {
		return nil, false, err
	}
	if expected == nil {
		return nil, false, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, b.services.Translator, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	data := b.applyBusinessLogic(req.Data)

	repo := b.repositories.TaskOutcome
	txCapable := b.services.Transactor != nil && b.services.Transactor.SupportsTransactions()
	if !txCapable {
		return uc.executeBestEffort(ctx, data, expected)
	}
	guard, guarded := isGuardedRepo(repo)
	writer, conditional := repo.(portsdomain.TaskOutcomeConditionalWriter)
	if !guarded || !conditional {
		return nil, false, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, b.services.Translator, "task_outcome.errors.guard_required", "[ERR-DEFAULT] task_outcome conditional update requires the cell-write guard and conditional writer on a transactional provider"))
	}

	var out *pb.TaskOutcome
	err := b.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		existing, rerr := repo.ReadTaskOutcome(txCtx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: data.Id}})
		if rerr != nil || existing == nil || len(existing.GetData()) == 0 {
			// Gone (deleted/foreign) since the caller's snapshot → conflict,
			// never a blind write.
			return portsdomain.ErrTaskOutcomeConflict
		}
		if err := guard.GuardCellWrite(txCtx, existing.GetData()[0].GetJobTaskId()); err != nil {
			return err
		}
		res, werr := writer.UpdateTaskOutcomeIfUnchanged(txCtx, data, expected)
		if werr != nil {
			return werr
		}
		out = res
		return nil
	})
	if errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &pb.UpdateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{out}}, false, nil
}

func (uc *UpdateTaskOutcomeIfUnchangedUseCase) executeBestEffort(ctx context.Context, data, expected *pb.TaskOutcome) (*pb.UpdateTaskOutcomeResponse, bool, error) {
	b := uc.base
	existing, err := b.repositories.TaskOutcome.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: data.Id}})
	if err != nil || existing == nil || len(existing.GetData()) == 0 {
		return nil, true, nil
	}
	if !SnapshotMatches(existing.GetData()[0], expected) {
		return nil, true, nil
	}
	resp, err := b.executeCore(ctx, &pb.UpdateTaskOutcomeRequest{Data: data}, data)
	return resp, false, err
}

// SnapshotMatches reports whether cur still carries expected's typed value
// (numeric_value, text_value, categorical_value, pass_fail_value),
// determination_note and date_modified — the in-memory mirror of the postgres
// adapter's IS NOT DISTINCT FROM predicate (task_outcome_conditional.go).
// fix3-backend (codex-review-impl3 #7): the typed non-numeric columns were
// missing here, so a stale snapshot differing only in text/categorical/
// pass-fail at the same timestamp was accepted by the non-transactional
// fallback.
func SnapshotMatches(cur, expected *pb.TaskOutcome) bool {
	if cur == nil || expected == nil {
		return cur == expected
	}
	return snapshotPtrEqual(cur.NumericValue, expected.NumericValue) &&
		snapshotPtrEqual(cur.TextValue, expected.TextValue) &&
		snapshotPtrEqual(cur.CategoricalValue, expected.CategoricalValue) &&
		snapshotPtrEqual(cur.PassFailValue, expected.PassFailValue) &&
		snapshotPtrEqual(cur.DeterminationNote, expected.DeterminationNote) &&
		snapshotPtrEqual(cur.DateModified, expected.DateModified)
}

// snapshotPtrEqual is SQL IS NOT DISTINCT FROM for optional proto scalars:
// both nil, or both set and equal.
func snapshotPtrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// CreateTaskOutcomeIfAbsentUseCase — create one outcome only when its
// (job_task_id, criteria_version_id) cell has no active outcome.
type CreateTaskOutcomeIfAbsentUseCase struct {
	base *CreateTaskOutcomeUseCase
}

// NewCreateTaskOutcomeIfAbsentUseCase shares the plain create use case's
// repositories/services (same gatekeeper, transactor, ID generator).
func NewCreateTaskOutcomeIfAbsentUseCase(repositories CreateTaskOutcomeRepositories, services CreateTaskOutcomeServices) *CreateTaskOutcomeIfAbsentUseCase {
	return &CreateTaskOutcomeIfAbsentUseCase{base: NewCreateTaskOutcomeUseCase(repositories, services)}
}

// Execute creates the outcome under the cell-write guard + job_task row lock.
func (uc *CreateTaskOutcomeIfAbsentUseCase) Execute(ctx context.Context, req *pb.CreateTaskOutcomeRequest) (*pb.CreateTaskOutcomeResponse, bool, error) {
	b := uc.base
	if err := b.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, false, err
	}
	if req == nil {
		return nil, false, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, b.services.Translator, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	if err := b.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, false, err
	}
	data := b.applyBusinessLogic(req.Data)

	repo := b.repositories.TaskOutcome
	txCapable := b.services.Transactor != nil && b.services.Transactor.SupportsTransactions()
	if !txCapable {
		resp, err := b.executeCore(ctx, req, data)
		return resp, false, err
	}
	guard, guarded := isGuardedRepo(repo)
	writer, conditional := repo.(portsdomain.TaskOutcomeConditionalWriter)
	if !guarded || !conditional {
		return nil, false, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, b.services.Translator, "task_outcome.errors.guard_required", "[ERR-DEFAULT] task_outcome create-if-absent requires the cell-write guard and conditional writer on a transactional provider"))
	}

	var out *pb.TaskOutcome
	err := b.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if err := guard.GuardCellWrite(txCtx, data.GetJobTaskId()); err != nil {
			return err
		}
		res, werr := writer.CreateTaskOutcomeIfAbsent(txCtx, data)
		if werr != nil {
			return werr
		}
		out = res
		return nil
	})
	if errors.Is(err, portsdomain.ErrTaskOutcomeConflict) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &pb.CreateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{out}}, false, nil
}
