package grade_compute

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// checkRecordGrant authorizes a recompute triggered by a recorded grade: the
// caller must hold task_outcome:create or task_outcome:update (the grants the
// grade-sheet save itself requires). Fails closed with the create denial.
func checkRecordGrant(ctx context.Context, gk *actiongate.ActionGatekeeper) error {
	createErr := gk.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.TaskOutcome, Action: entityid.ActionCreate})
	if createErr == nil {
		return nil
	}
	if err := gk.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.TaskOutcome, Action: entityid.ActionUpdate}); err == nil {
		return nil
	}
	return createErr
}
