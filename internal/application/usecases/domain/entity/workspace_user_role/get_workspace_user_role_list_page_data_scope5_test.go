package workspace_user_role

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

type scope5WURRepo struct {
	workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	byWorkspace map[string][]*workspaceuserrolepb.WorkspaceUserRole
	pageCalls   int
	listCalls   int
}

func (r *scope5WURRepo) ListWorkspaceUserRoles(context.Context, *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
	r.listCalls++
	return nil, fmt.Errorf("generic list is unscoped")
}

func (r *scope5WURRepo) GetWorkspaceUserRoleListPageData(ctx context.Context, req *workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest) (*workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse, error) {
	r.pageCalls++
	actor, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetFilters() != nil || req.GetSearch() != nil || req.GetSort() != nil {
		return nil, fmt.Errorf("page source must retain default order without prefiltering")
	}
	limit, page := int(req.GetPagination().GetLimit()), int(req.GetPagination().GetOffset().GetPage())
	if limit != 100 || page < 1 {
		return nil, fmt.Errorf("unexpected page bounds: limit=%d page=%d", limit, page)
	}
	rows := r.byWorkspace[actor.WorkspaceID]
	start := (page - 1) * limit
	if start > len(rows) {
		start = len(rows)
	}
	end := min(start+limit, len(rows))
	return &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse{
		WorkspaceUserRoleList: rows[start:end],
		Pagination: &commonpb.PaginationResponse{
			TotalItems: int32(len(rows)),
			HasNext:    end < len(rows),
		},
		Success: true,
	}, nil
}

func scope5UseCase(repo *scope5WURRepo) *GetWorkspaceUserRoleListPageDataUseCase {
	return NewGetWorkspaceUserRoleListPageDataUseCase(
		GetWorkspaceUserRoleListPageDataRepositories{WorkspaceUserRole: repo},
		GetWorkspaceUserRoleListPageDataServices{ActionGatekeeper: actiongate.NewActionGatekeeper(tbAllowAuthorizer{}, nil)},
	)
}

func scope5Context(workspace string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "admin", WorkspaceID: workspace})
}

func TestGetWorkspaceUserRoleListPageDataScope5ForeignRowsExcludedAndSemantics(t *testing.T) {
	repo := &scope5WURRepo{byWorkspace: map[string][]*workspaceuserrolepb.WorkspaceUserRole{
		"ws-a": {
			{Id: "a1", RoleId: "needle-a", Active: true},
			{Id: "a2", RoleId: "other", Active: true},
			{Id: "a3", RoleId: "needle-c", Active: true},
			{Id: "a4", RoleId: "needle-b", Active: true},
		},
		"ws-b": {{Id: "foreign", RoleId: "needle-z", Active: true}},
	}}
	uc := scope5UseCase(repo)
	resp, err := uc.Execute(scope5Context("ws-a"), &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 2}}},
		Filters:    &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{Field: "active", FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: true}}}}},
		Sort:       &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id", Direction: commonpb.SortDirection_DESC}}},
		Search:     &commonpb.SearchRequest{Query: "needle", Options: &commonpb.SearchOptions{SearchFields: []string{"role_id"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, row := range resp.WorkspaceUserRoleList {
		ids = append(ids, row.Id)
	}
	if !slices.Equal(ids, []string{"a1"}) || !resp.Success || resp.Pagination.GetTotalItems() != 3 || resp.Pagination.GetCurrentPage() != 2 || resp.Pagination.HasNext || !resp.Pagination.HasPrev || len(resp.SearchResults) != 1 || resp.SearchResults[0].Score <= 0 {
		t.Fatalf("scoped filter/search/sort/page response: ids=%v pagination=%v search=%v", ids, resp.Pagination, resp.SearchResults)
	}
	if repo.listCalls != 0 || repo.pageCalls != 1 {
		t.Fatalf("repository calls: generic=%d scoped=%d", repo.listCalls, repo.pageCalls)
	}
}

func TestGetWorkspaceUserRoleListPageDataScope5ReadsAllScopedPages(t *testing.T) {
	rows := make([]*workspaceuserrolepb.WorkspaceUserRole, 105)
	for i := range rows {
		rows[i] = &workspaceuserrolepb.WorkspaceUserRole{Id: fmt.Sprintf("a%03d", i+1), Active: true}
	}
	repo := &scope5WURRepo{byWorkspace: map[string][]*workspaceuserrolepb.WorkspaceUserRole{
		"ws-a": rows,
		"ws-b": {{Id: "foreign", Active: true}},
	}}
	resp, err := scope5UseCase(repo).Execute(scope5Context("ws-a"), &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{Limit: 10, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 11}}},
		Sort:       &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, row := range resp.WorkspaceUserRoleList {
		ids = append(ids, row.Id)
	}
	if !slices.Equal(ids, []string{"a101", "a102", "a103", "a104", "a105"}) || resp.Pagination.GetTotalItems() != 105 || repo.pageCalls != 2 || repo.listCalls != 0 {
		t.Fatalf("full scoped set: ids=%v total=%d scoped calls=%d generic calls=%d", ids, resp.Pagination.GetTotalItems(), repo.pageCalls, repo.listCalls)
	}
}

func TestGetWorkspaceUserRoleListPageDataScope5RequiresWorkspaceBeforeRepository(t *testing.T) {
	for _, ctx := range []context.Context{context.Background(), scope5Context("")} {
		repo := &scope5WURRepo{}
		if _, err := scope5UseCase(repo).Execute(ctx, &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{}); err == nil {
			t.Fatal("missing workspace must fail")
		}
		if repo.pageCalls != 0 || repo.listCalls != 0 {
			t.Fatalf("missing workspace queried repository: scoped=%d generic=%d", repo.pageCalls, repo.listCalls)
		}
	}
}

func TestGetWorkspaceUserRoleListPageDataScope5EmptyScopedSet(t *testing.T) {
	repo := &scope5WURRepo{byWorkspace: map[string][]*workspaceuserrolepb.WorkspaceUserRole{
		"ws-b": {{Id: "foreign", Active: true}},
	}}
	resp, err := scope5UseCase(repo).Execute(scope5Context("ws-a"), &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || len(resp.WorkspaceUserRoleList) != 0 || len(resp.SearchResults) != 0 || resp.Pagination.GetTotalItems() != 0 || resp.Pagination.GetCurrentPage() != 1 || resp.Pagination.GetTotalPages() != 1 {
		t.Fatalf("empty scoped response: %+v", resp)
	}
	if repo.pageCalls != 1 || repo.listCalls != 0 {
		t.Fatalf("repository calls: scoped=%d generic=%d", repo.pageCalls, repo.listCalls)
	}
}

func TestGetWorkspaceUserRoleListPageDataScope5DefaultOrder(t *testing.T) {
	repo := &scope5WURRepo{byWorkspace: map[string][]*workspaceuserrolepb.WorkspaceUserRole{
		"ws-a": {
			{Id: "a3", Active: true},
			{Id: "a1", Active: true},
			{Id: "a2", Active: true},
		},
	}}
	resp, err := scope5UseCase(repo).Execute(scope5Context("ws-a"), &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, row := range resp.WorkspaceUserRoleList {
		ids = append(ids, row.Id)
	}
	if !slices.Equal(ids, []string{"a3", "a1"}) || resp.Pagination.GetTotalItems() != 3 {
		t.Fatalf("default adapter order and pagination: ids=%v total=%d", ids, resp.Pagination.GetTotalItems())
	}
}
