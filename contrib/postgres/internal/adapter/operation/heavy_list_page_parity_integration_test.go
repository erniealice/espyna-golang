//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	josumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	possumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	ttcpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// Plan 20260927-db-query-performance AC-09 (audit DB-07). The six list pages on
// high-volume tables moved from COUNT(*) OVER () to the Q-PAGE-COUNT heavyweight
// tier (ids-first page + scalar count). Each case compares the adapter's ids,
// order, and total with the legacy window-count statement over the same filters
// (admin principal, empty search, default sort). SELECT-only; skips without
// TEST_DATABASE_URL.

type heavyListCase struct {
	name    string
	legacy  string // $1 = limit, $2 = offset, $3 = workspace when used
	usesWS  bool
	fetch   func(ctx context.Context, page int32) (ids []string, total int32, err error)
}

func TestHeavyListPagesDB_Parity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	var ws string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC, workspace_id LIMIT 1`,
	).Scan(&ws); err != nil {
		t.Skipf("no jobs: %v", err)
	}
	admin := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: ws})
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	pg := func(n int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 50, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}
	}
	order := func(cols []string, fallback string) string {
		clause, err := postgresCore.BuildOrderBy(cols, nil, fallback)
		if err != nil {
			t.Fatalf("order: %v", err)
		}
		return clause
	}

	taskRepo := NewPostgresJobTaskRepository(ops, "job_task").(*PostgresJobTaskRepository)
	phaseRepo := NewPostgresJobPhaseRepository(ops, "job_phase").(*PostgresJobPhaseRepository)
	outcomeRepo := NewPostgresTaskOutcomeRepository(ops, "task_outcome").(*PostgresTaskOutcomeRepository)
	posRepo := NewPostgresPhaseOutcomeSummaryRepository(ops, "phase_outcome_summary").(*PostgresPhaseOutcomeSummaryRepository)
	josRepo := NewPostgresJobOutcomeSummaryRepository(ops, "job_outcome_summary").(*PostgresJobOutcomeSummaryRepository)
	ttcRepo := NewPostgresTemplateTaskCriteriaRepository(ops, "template_task_criteria").(*PostgresTemplateTaskCriteriaRepository)

	cases := []heavyListCase{
		{
			name:   "job_task",
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.JobTask + ` e WHERE e.active = true ` + order(jobTaskSortableSQLCols, "step_order ASC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := taskRepo.GetJobTaskListPageData(ctx, &jobtaskpb.GetJobTaskListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetJobTaskList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
		{
			name:   "job_phase",
			usesWS: true,
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.JobPhase + ` e JOIN ` + entityid.Job + ` j ON j.id = e.job_id AND j.workspace_id = $3 WHERE e.active = true ` + order(jobPhaseSortableSQLCols, "phase_order ASC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := phaseRepo.GetJobPhaseListPageData(ctx, &jobphasepb.GetJobPhaseListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetJobPhaseList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
		{
			name:   "task_outcome",
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.TaskOutcome + ` e WHERE e.active = true ` + order(taskOutcomeSortableSQLCols, "date_created DESC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := outcomeRepo.GetTaskOutcomeListPageData(ctx, &taskoutcomepb.GetTaskOutcomeListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetTaskOutcomeList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
		{
			name:   "phase_outcome_summary",
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.PhaseOutcomeSummary + ` e WHERE e.active = true ` + order(phaseOutcomeSummarySortableSQLCols, "date_created DESC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := posRepo.GetPhaseOutcomeSummaryListPageData(ctx, &possumpb.GetPhaseOutcomeSummaryListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetPhaseOutcomeSummaryList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
		{
			name:   "job_outcome_summary",
			usesWS: true,
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.JobOutcomeSummary + ` e WHERE e.active = true AND e.workspace_id = $3 ` + order(jobOutcomeSummarySortableSQLCols, "date_created DESC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := josRepo.GetJobOutcomeSummaryListPageData(ctx, &josumpb.GetJobOutcomeSummaryListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetJobOutcomeSummaryList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
		{
			name:   "template_task_criteria",
			legacy: `SELECT e.id, COUNT(*) OVER () FROM ` + entityid.TemplateTaskCriteria + ` e WHERE e.active = true ` + order(templateTaskCriteriaSortableSQLCols, "sequence_order ASC") + ` LIMIT $1 OFFSET $2`,
			fetch: func(ctx context.Context, p int32) ([]string, int32, error) {
				r, err := ttcRepo.GetTemplateTaskCriteriaListPageData(ctx, &ttcpb.GetTemplateTaskCriteriaListPageDataRequest{Pagination: pg(p)})
				var ids []string
				for _, x := range r.GetTemplateTaskCriteriaList() {
					ids = append(ids, x.GetId())
				}
				return ids, r.GetPagination().GetTotalItems(), err
			},
		},
	}

	for _, tc := range cases {
		for _, page := range []int32{1, 21} {
			t.Run(tc.name+"/page_"+string(rune('0'+page/10))+string(rune('0'+page%10)), func(t *testing.T) {
				args := []any{int32(50), (page - 1) * 50}
				if tc.usesWS {
					args = append(args, ws)
				}
				wantIDs, wantTotal := legacyIDsAndTotal(t, db, tc.legacy, args)
				gotIDs, gotTotal, err := tc.fetch(admin, page)
				if err != nil {
					t.Fatalf("adapter: %v", err)
				}
				if !slices.Equal(gotIDs, wantIDs) {
					t.Fatalf("ids/order differ: new=%d rows, legacy=%d rows", len(gotIDs), len(wantIDs))
				}
				if len(wantIDs) > 0 && int64(gotTotal) != wantTotal {
					t.Fatalf("total: new=%d legacy=%d", gotTotal, wantTotal)
				}
				t.Logf("rows=%d total=%d", len(gotIDs), gotTotal)
			})
		}
	}
}

func legacyIDsAndTotal(t *testing.T, db *sql.DB, stmt string, args []any) ([]string, int64) {
	t.Helper()
	rows, err := db.Query(stmt, args...)
	if err != nil {
		t.Fatalf("legacy query: %v", err)
	}
	defer rows.Close()
	var ids []string
	var total int64
	for rows.Next() {
		var id string
		if err := rows.Scan(&id, &total); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	return ids, total
}
