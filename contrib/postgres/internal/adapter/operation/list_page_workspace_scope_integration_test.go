//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// These probes use committed rows only. TEST_DATABASE_URL must enforce
// default_transaction_read_only=on for every connection in the pool.
func requireReadOnlyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestDB(t)
	var mode string
	if err := db.QueryRow(`SHOW default_transaction_read_only`).Scan(&mode); err != nil || mode != "on" {
		db.Close()
		t.Fatalf("TEST_DATABASE_URL must set default_transaction_read_only=on: mode=%q err=%v", mode, err)
	}
	return db
}

func listScopePage(n int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{Limit: 50, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}
}

func TestListPageWorkspaceScope_LiveCloneAdminStaffParity(t *testing.T) {
	db := requireReadOnlyTestDB(t)
	defer db.Close()
	ctx := context.Background()
	var ws, taskStaff, outcomeStaff string
	if err := db.QueryRow(`SELECT workspace_id FROM job GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&ws); err != nil {
		t.Skipf("no workspace jobs: %v", err)
	}
	if err := db.QueryRow(`SELECT assigned_to FROM job_task WHERE assigned_to IS NOT NULL GROUP BY assigned_to ORDER BY count(*) DESC LIMIT 1`).Scan(&taskStaff); err != nil {
		t.Skipf("no assigned tasks: %v", err)
	}
	if err := db.QueryRow(`SELECT recorded_by FROM task_outcome WHERE recorded_by IS NOT NULL GROUP BY recorded_by ORDER BY count(*) DESC LIMIT 1`).Scan(&outcomeStaff); err != nil {
		t.Skipf("no recorded outcomes: %v", err)
	}
	taskRepo := NewPostgresJobTaskRepository(postgresCore.NewWorkspaceAwareOperations(db), "job_task").(*PostgresJobTaskRepository)
	outcomeRepo := NewPostgresTaskOutcomeRepository(postgresCore.NewWorkspaceAwareOperations(db), "task_outcome").(*PostgresTaskOutcomeRepository)
	for _, principal := range []struct {
		name         string
		staff        bool
		staffForTask string
		staffForOut  string
	}{
		{name: "admin"},
		{name: "staff", staff: true, staffForTask: taskStaff, staffForOut: outcomeStaff},
	} {
		for _, page := range []int32{1, 21} {
			t.Run(principal.name+"/job_task/page_"+string(rune('0'+page/10))+string(rune('0'+page%10)), func(t *testing.T) {
				id := &identity.RequestIdentity{WorkspaceID: ws}
				args := []any{50, (page - 1) * 50, ws}
				legacy := `SELECT jt.id, count(*) OVER () FROM job_task jt JOIN job_phase jp ON jp.id=jt.job_phase_id JOIN job j ON j.id=jp.job_id WHERE jt.active=true AND j.workspace_id=$3`
				if principal.staff {
					id.PrincipalType, id.PrincipalID = 7, principal.staffForTask
					legacy += ` AND jt.assigned_to=$4`
					args = append(args, principal.staffForTask)
				}
				legacy += ` ORDER BY jt.step_order ASC, jt.id ASC LIMIT $1 OFFSET $2`
				wantIDs, wantTotal := legacyIDsAndTotal(t, db, legacy, args)
				got, err := taskRepo.GetJobTaskListPageData(identity.WithRequestIdentity(ctx, id), &jobtaskpb.GetJobTaskListPageDataRequest{Pagination: listScopePage(page)})
				if err != nil {
					t.Fatal(err)
				}
				var gotIDs []string
				for _, x := range got.GetJobTaskList() {
					gotIDs = append(gotIDs, x.GetId())
				}
				if !slices.Equal(gotIDs, wantIDs) || len(wantIDs) > 0 && int64(got.GetPagination().GetTotalItems()) != wantTotal {
					t.Fatalf("ids/order/total differ: got=%d/%d want=%d/%d", len(gotIDs), got.GetPagination().GetTotalItems(), len(wantIDs), wantTotal)
				}
				t.Logf("rows=%d total=%d", len(gotIDs), got.GetPagination().GetTotalItems())
			})
			t.Run(principal.name+"/task_outcome/page_"+string(rune('0'+page/10))+string(rune('0'+page%10)), func(t *testing.T) {
				id := &identity.RequestIdentity{WorkspaceID: ws}
				args := []any{50, (page - 1) * 50, ws}
				legacy := `SELECT o.id, count(*) OVER () FROM task_outcome o JOIN job_task jt ON jt.id=o.job_task_id JOIN job_phase jp ON jp.id=jt.job_phase_id JOIN job j ON j.id=jp.job_id WHERE o.active=true AND j.workspace_id=$3`
				if principal.staff {
					id.PrincipalType, id.PrincipalID = 7, principal.staffForOut
					legacy += ` AND (o.recorded_by=$4 OR o.reviewed_by=$4)`
					args = append(args, principal.staffForOut)
				}
				legacy += ` ORDER BY o.date_created DESC, o.id ASC LIMIT $1 OFFSET $2`
				wantIDs, wantTotal := legacyIDsAndTotal(t, db, legacy, args)
				got, err := outcomeRepo.GetTaskOutcomeListPageData(identity.WithRequestIdentity(ctx, id), &taskoutcomepb.GetTaskOutcomeListPageDataRequest{Pagination: listScopePage(page)})
				if err != nil {
					t.Fatal(err)
				}
				var gotIDs []string
				for _, x := range got.GetTaskOutcomeList() {
					gotIDs = append(gotIDs, x.GetId())
				}
				if !slices.Equal(gotIDs, wantIDs) || len(wantIDs) > 0 && int64(got.GetPagination().GetTotalItems()) != wantTotal {
					t.Fatalf("ids/order/total differ: got=%d/%d want=%d/%d", len(gotIDs), got.GetPagination().GetTotalItems(), len(wantIDs), wantTotal)
				}
				t.Logf("rows=%d total=%d", len(gotIDs), got.GetPagination().GetTotalItems())
			})
		}
	}
}

func TestListPageWorkspaceScope_LiveForeignWorkspaceExcluded(t *testing.T) {
	db := requireReadOnlyTestDB(t)
	defer db.Close()
	ctx := context.Background()
	var owned, foreign string
	if err := db.QueryRow(`SELECT j.workspace_id FROM job_task jt JOIN job_phase jp ON jp.id=jt.job_phase_id JOIN job j ON j.id=jp.job_id WHERE jt.active=true GROUP BY j.workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&owned); err != nil {
		t.Skipf("no job tasks: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM workspace WHERE id<>$1 ORDER BY id LIMIT 1`, owned).Scan(&foreign); err != nil {
		t.Skipf("no second workspace: %v", err)
	}
	var foreignRows int
	if err := db.QueryRow(`SELECT count(*) FROM job_task jt JOIN job_phase jp ON jp.id=jt.job_phase_id JOIN job j ON j.id=jp.job_id WHERE jt.active=true AND j.workspace_id=$1`, foreign).Scan(&foreignRows); err != nil {
		t.Fatal(err)
	}
	if foreignRows != 0 {
		t.Skipf("foreign workspace has %d tasks; choose a clean lane", foreignRows)
	}
	repo := NewPostgresJobTaskRepository(postgresCore.NewWorkspaceAwareOperations(db), "job_task").(*PostgresJobTaskRepository)
	for _, tc := range []struct {
		ws       string
		wantRows bool
	}{{owned, true}, {foreign, false}} {
		result, err := repo.GetJobTaskListPageData(identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: tc.ws}), &jobtaskpb.GetJobTaskListPageDataRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if (len(result.GetJobTaskList()) > 0) != tc.wantRows {
			t.Fatalf("workspace returned %d rows, wantRows=%t", len(result.GetJobTaskList()), tc.wantRows)
		}
		t.Logf("workspace has_rows=%t page_rows=%d total=%d", tc.wantRows, len(result.GetJobTaskList()), result.GetPagination().GetTotalItems())
	}
}
