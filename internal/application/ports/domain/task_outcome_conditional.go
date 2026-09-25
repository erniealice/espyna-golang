package domain

import (
	"context"
	"errors"

	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// ErrTaskOutcomeConflict is the CONFLICT sentinel of the two conditional
// task_outcome writes (docs/plan/20260925-criterion-descriptors-by-program-
// year, schema-proposal.md §9.1 / interfaces.md §2b, Q26). It means the cell
// changed (or appeared) between the caller's snapshot read and the write: the
// write was NOT applied and value + note are exactly as another writer left
// them. Callers reject the cell visibly (fayna: `cell_changed_retry`); they
// never retry blindly.
var ErrTaskOutcomeConflict = errors.New("CONFLICT: task outcome changed since it was read")

// TaskOutcomeConditionalWriter is the hand-written conditional-write port for
// the grade-sheet save path. It is NOT part of the generated
// TaskOutcomeDomainServiceServer contract (same pattern as
// RatingDescriptionSetLifecycleRepository): the use cases type-assert the
// injected repository against it, so no esqyma RPC / provider-wide surface is
// required. Both methods require an ambient transaction and a trusted
// workspace on ctx, failing closed otherwise.
type TaskOutcomeConditionalWriter interface {
	// UpdateTaskOutcomeIfUnchanged applies ONE conditional UPDATE of the
	// outcome's value/type/note columns, matching id + trusted workspace +
	// active AND numeric_value / determination_note / date_modified IS NOT
	// DISTINCT FROM the expected snapshot. 0 rows → ErrTaskOutcomeConflict.
	UpdateTaskOutcomeIfUnchanged(ctx context.Context, data *taskoutcomepb.TaskOutcome, expected *taskoutcomepb.TaskOutcome) (*taskoutcomepb.TaskOutcome, error)
	// CreateTaskOutcomeIfAbsent locks the owning job_task row FOR UPDATE and
	// inserts only when no active outcome exists for (job_task_id,
	// criteria_version_id); otherwise ErrTaskOutcomeConflict.
	CreateTaskOutcomeIfAbsent(ctx context.Context, data *taskoutcomepb.TaskOutcome) (*taskoutcomepb.TaskOutcome, error)
}
