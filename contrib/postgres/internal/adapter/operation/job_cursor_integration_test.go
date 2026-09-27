//go:build postgresql

package operation

import (
	"context"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

func TestJobCursor_ReadOnlyParity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	var ws string
	if err := db.QueryRowContext(ctx, `SELECT workspace_id FROM job WHERE active AND name ILIKE '%10%'
		GROUP BY workspace_id HAVING count(*) BETWEEN 21 AND 200 ORDER BY count(*) DESC LIMIT 1`).Scan(&ws); err != nil {
		t.Skipf("no bounded multi-page job sample: %v", err)
	}
	caller := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: ws})
	repo := NewPostgresJobRepository(postgresCore.NewWorkspaceAwareOperations(db), "job").(*PostgresJobRepository)
	page := func(n int32) *pb.GetJobListPageDataRequest {
		return &pb.GetJobListPageDataRequest{Search: &commonpb.SearchRequest{Query: "10"}, Pagination: &commonpb.PaginationRequest{Limit: 20, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}}
	}
	cursor := func(token string) *pb.GetJobListPageDataRequest {
		return &pb.GetJobListPageDataRequest{Search: &commonpb.SearchRequest{Query: "10"}, Pagination: &commonpb.PaginationRequest{Limit: 20, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}}
	}
	ids := func(r *pb.GetJobListPageDataResponse) []string {
		var out []string
		for _, row := range r.GetJobList() {
			out = append(out, row.GetId())
		}
		return out
	}
	var want [][]string
	for n := int32(1); ; n++ {
		r, err := repo.GetJobListPageData(caller, page(n))
		if err != nil {
			t.Fatal(err)
		}
		if len(r.GetJobList()) == 0 {
			break
		}
		want = append(want, ids(r))
	}
	if len(want) < 2 {
		t.Skipf("only %d pages", len(want))
	}
	r, err := repo.GetJobListPageData(caller, page(1))
	if err != nil {
		t.Fatal(err)
	}
	for i, expected := range want {
		if !slices.Equal(ids(r), expected) {
			t.Fatalf("next page %d: got %v want %v", i+1, ids(r), expected)
		}
		if i+1 < len(want) {
			r, err = repo.GetJobListPageData(caller, cursor(r.GetPagination().GetNextCursor()))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := len(want) - 2; i >= 0; i-- {
		r, err = repo.GetJobListPageData(caller, cursor(r.GetPagination().GetPrevCursor()))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(ids(r), want[i]) {
			t.Fatalf("prev page %d: got %v want %v", i+1, ids(r), want[i])
		}
	}
	t.Logf("%d rows, %d pages; Next/Prev IDs and order match offset", r.GetPagination().GetTotalItems(), len(want))
}
