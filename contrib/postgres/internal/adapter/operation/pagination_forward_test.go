//go:build postgresql

package operation

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	templatetaskcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// M8 — pagination-forwarding regression (ADAPTER LIST ROW-CAP lesson). The
// silent-truncation fix (forward req.Pagination into ListParams, not just
// req.Filters) landed identically in three adapters; each is a one-line
// condition a mechanical simplification could revert. These fakes assert the
// caller's *Pagination pointer reaches dbOps.List so a regression fails loudly.

// listCapturingDBOps records the *ListParams handed to List and returns an empty
// result. (Distinct from recordingDBOps so the existing H1 tests are untouched.)
type listCapturingDBOps struct {
	lastListParams *interfaces.ListParams
}

func (f *listCapturingDBOps) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *listCapturingDBOps) Read(context.Context, string, string) (map[string]any, error) {
	return nil, nil
}
func (f *listCapturingDBOps) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *listCapturingDBOps) Delete(context.Context, string, string) error     { return nil }
func (f *listCapturingDBOps) HardDelete(context.Context, string, string) error { return nil }
func (f *listCapturingDBOps) List(_ context.Context, _ string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	f.lastListParams = params
	return &interfaces.ListResult{}, nil
}
func (f *listCapturingDBOps) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (f *listCapturingDBOps) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}

func TestListJobTasks_ForwardsPagination(t *testing.T) {
	fake := &listCapturingDBOps{}
	r := NewPostgresJobTaskRepository(fake, "job_task")
	pag := &commonpb.PaginationRequest{Limit: 25}

	if _, err := r.ListJobTasks(context.Background(), &jobtaskpb.ListJobTasksRequest{Pagination: pag}); err != nil {
		t.Fatalf("ListJobTasks error: %v", err)
	}
	assertPaginationForwarded(t, "job_task", fake.lastListParams, pag)
}

func TestListTaskOutcomes_ForwardsPagination(t *testing.T) {
	fake := &listCapturingDBOps{}
	r := NewPostgresTaskOutcomeRepository(fake, "task_outcome")
	pag := &commonpb.PaginationRequest{Limit: 25}

	if _, err := r.ListTaskOutcomes(context.Background(), &taskoutcomepb.ListTaskOutcomesRequest{Pagination: pag}); err != nil {
		t.Fatalf("ListTaskOutcomes error: %v", err)
	}
	assertPaginationForwarded(t, "task_outcome", fake.lastListParams, pag)
}

func TestListTemplateTaskCriterias_ForwardsPagination(t *testing.T) {
	fake := &listCapturingDBOps{}
	r := NewPostgresTemplateTaskCriteriaRepository(fake, "template_task_criteria")
	pag := &commonpb.PaginationRequest{Limit: 25}

	if _, err := r.ListTemplateTaskCriterias(context.Background(), &templatetaskcriteriapb.ListTemplateTaskCriteriasRequest{Pagination: pag}); err != nil {
		t.Fatalf("ListTemplateTaskCriterias error: %v", err)
	}
	assertPaginationForwarded(t, "template_task_criteria", fake.lastListParams, pag)
}

// TestListJobTasks_NilParamsWhenNoFiltersOrPagination pins the other half of the
// contract: omitting both Filters and Pagination keeps the nil-params behavior.
func TestListJobTasks_NilParamsWhenEmpty(t *testing.T) {
	fake := &listCapturingDBOps{}
	r := NewPostgresJobTaskRepository(fake, "job_task")
	if _, err := r.ListJobTasks(context.Background(), &jobtaskpb.ListJobTasksRequest{}); err != nil {
		t.Fatalf("ListJobTasks error: %v", err)
	}
	if fake.lastListParams != nil {
		t.Errorf("expected nil ListParams when no Filters/Pagination, got %+v", fake.lastListParams)
	}
}

func assertPaginationForwarded(t *testing.T, entity string, got *interfaces.ListParams, want *commonpb.PaginationRequest) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: List received nil ListParams — pagination dropped (row-cap regression)", entity)
	}
	if got.Pagination != want {
		t.Errorf("%s: List must forward the caller's *Pagination pointer; got %p want %p", entity, got.Pagination, want)
	}
}
