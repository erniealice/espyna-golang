package grade_compute

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/shared/identity"
)

type recordGrantAuthorizer struct{ allowed map[string]bool }

func (f *recordGrantAuthorizer) IsEnabled() bool { return true }
func (f *recordGrantAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

func recordGrantCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "teacher-1", WorkspaceID: "ws-1"})
}

func servicesWith(perms ...string) Services {
	allowed := map[string]bool{}
	for _, p := range perms {
		allowed[p] = true
	}
	return Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&recordGrantAuthorizer{allowed: allowed}, nil)}
}

// isAuthzDenial reports whether err came from the gate (the body is never
// reached). A nil request past the gate fails validation instead.
func isAuthzDenial(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "denied")
}

func TestExecuteAfterRecordAuthorizedByRecordGrant(t *testing.T) {
	for _, perms := range [][]string{{"task_outcome:create"}, {"task_outcome:update"}} {
		phase := NewComputePhaseOutcomeUseCase(Repositories{}, servicesWith(perms...))
		if _, err := phase.ExecuteAfterRecord(recordGrantCtx(), nil); err == nil || isAuthzDenial(err) {
			t.Fatalf("phase with %v: err = %v, want the post-gate validation error", perms, err)
		}
		job := NewComputeJobOutcomeUseCase(Repositories{}, servicesWith(perms...))
		if _, err := job.ExecuteAfterRecord(recordGrantCtx(), nil); err == nil || isAuthzDenial(err) {
			t.Fatalf("job with %v: err = %v, want the post-gate validation error", perms, err)
		}
	}
}

func TestExecuteAfterRecordDeniedWithoutRecordGrant(t *testing.T) {
	svc := servicesWith("task_outcome:list", "task_outcome:read")
	if _, err := NewComputePhaseOutcomeUseCase(Repositories{}, svc).ExecuteAfterRecord(recordGrantCtx(), nil); !isAuthzDenial(err) {
		t.Fatalf("phase err = %v, want an authorization denial", err)
	}
	if _, err := NewComputeJobOutcomeUseCase(Repositories{}, svc).ExecuteAfterRecord(recordGrantCtx(), nil); !isAuthzDenial(err) {
		t.Fatalf("job err = %v, want an authorization denial", err)
	}
}

func TestExecuteStillRequiresSummaryGrant(t *testing.T) {
	svc := servicesWith("task_outcome:create", "task_outcome:update")
	if _, err := NewComputePhaseOutcomeUseCase(Repositories{}, svc).Execute(recordGrantCtx(), nil); !isAuthzDenial(err) {
		t.Fatalf("phase Execute err = %v, want denial without phase_outcome_summary:create", err)
	}
	if _, err := NewComputeJobOutcomeUseCase(Repositories{}, svc).Execute(recordGrantCtx(), nil); !isAuthzDenial(err) {
		t.Fatalf("job Execute err = %v, want denial without job_outcome_summary:create", err)
	}
}
