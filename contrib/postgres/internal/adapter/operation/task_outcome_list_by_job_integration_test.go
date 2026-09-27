//go:build postgresql

package operation

import (
	"context"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Plan 20260927-db-query-performance AC-03 (audit DB-11). ListByJob used to
// filter jt.job_id, a column job_task does not have, so every call failed with
// UndefinedColumn. A task reaches its job through job_phase, and the job must
// sit in the trusted-context workspace. SELECT-only; skips without
// TEST_DATABASE_URL.

func TestTaskOutcomeListByJob_JoinsPhase(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	var jobID, workspaceID string
	var want int
	err := db.QueryRowContext(ctx, `
		SELECT j.id, j.workspace_id, count(*)
		FROM `+entityid.TaskOutcome+` to_
		JOIN `+entityid.JobTask+` jt ON to_.job_task_id = jt.id
		JOIN `+entityid.JobPhase+` jp ON jt.job_phase_id = jp.id
		JOIN `+entityid.Job+` j ON jp.job_id = j.id
		WHERE to_.active = true
		GROUP BY j.id, j.workspace_id
		ORDER BY count(*) DESC, j.id
		LIMIT 1`).Scan(&jobID, &workspaceID, &want)
	if err != nil {
		t.Skipf("no job with active task outcomes in TEST_DATABASE_URL: %v", err)
	}

	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome")
	wsCtx := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspaceID})
	resp, err := repo.ListByJob(wsCtx, &pb.ListTaskOutcomesByJobRequest{JobId: jobID})
	if err != nil {
		t.Fatalf("ListByJob: %v", err)
	}
	if got := len(resp.GetTaskOutcomes()); got != want {
		t.Fatalf("ListByJob returned %d outcomes, want %d", got, want)
	}
}

func TestTaskOutcomeListByJob_WorkspaceScoped(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	var jobID string
	if err := db.QueryRowContext(ctx, `
		SELECT jp.job_id
		FROM `+entityid.TaskOutcome+` to_
		JOIN `+entityid.JobTask+` jt ON to_.job_task_id = jt.id
		JOIN `+entityid.JobPhase+` jp ON jt.job_phase_id = jp.id
		WHERE to_.active = true
		ORDER BY jp.job_id
		LIMIT 1`).Scan(&jobID); err != nil {
		t.Skipf("no job with active task outcomes in TEST_DATABASE_URL: %v", err)
	}

	repo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome")

	foreign := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: "not-this-workspace"})
	resp, err := repo.ListByJob(foreign, &pb.ListTaskOutcomesByJobRequest{JobId: jobID})
	if err != nil {
		t.Fatalf("ListByJob foreign workspace: %v", err)
	}
	if n := len(resp.GetTaskOutcomes()); n != 0 {
		t.Fatalf("foreign workspace saw %d outcomes, want 0", n)
	}

	resp, err = repo.ListByJob(ctx, &pb.ListTaskOutcomesByJobRequest{JobId: jobID})
	if err != nil {
		t.Fatalf("ListByJob without identity: %v", err)
	}
	if n := len(resp.GetTaskOutcomes()); n != 0 {
		t.Fatalf("missing workspace saw %d outcomes, want 0 (fail closed)", n)
	}
}
