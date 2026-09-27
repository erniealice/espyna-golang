//go:build postgresql

package subscription

import (
	"context"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

func TestPricePlanCursor_ReadOnlyParityAndForeignBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("local read-only database")
	}
	db := openRelationPaginationDB(t)
	defer db.Close()
	ctx := context.Background()
	var workspace, foreignID string
	if err := db.QueryRowContext(ctx, `SELECT pl.workspace_id FROM price_plan pp JOIN plan pl ON pl.id = pp.plan_id
		WHERE pp.active GROUP BY pl.workspace_id HAVING count(*) >= 8 AND count(*) = count(*) FILTER (
		WHERE pp.id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
		ORDER BY count(*) DESC LIMIT 1`).Scan(&workspace); err != nil {
		t.Skipf("no multi-page workspace: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT pp.id FROM price_plan pp JOIN plan pl ON pl.id = pp.plan_id
		WHERE pp.active AND pl.workspace_id <> $1 AND pp.id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
		ORDER BY pp.id LIMIT 1`, workspace).Scan(&foreignID); err != nil {
		t.Skipf("no foreign price plan: %v", err)
	}
	caller := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspace})
	repo := NewPostgresPricePlanRepository(postgresCore.NewWorkspaceAwareOperations(db), "price_plan").(*PostgresPricePlanRepository)
	page := func(n int32) *pb.GetPricePlanListPageDataRequest {
		return &pb.GetPricePlanListPageDataRequest{Pagination: &commonpb.PaginationRequest{Limit: 7, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}}
	}
	cursor := func(token string) *pb.GetPricePlanListPageDataRequest {
		return &pb.GetPricePlanListPageDataRequest{Pagination: &commonpb.PaginationRequest{Limit: 7, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}}
	}
	ids := func(r *pb.GetPricePlanListPageDataResponse) []string {
		var out []string
		for _, row := range r.GetPricePlanList() {
			out = append(out, row.GetId())
		}
		return out
	}
	var want [][]string
	for n := int32(1); ; n++ {
		r, err := repo.GetPricePlanListPageData(caller, page(n))
		if err != nil {
			t.Fatal(err)
		}
		if len(r.GetPricePlanList()) == 0 {
			break
		}
		want = append(want, ids(r))
	}
	if len(want) < 2 {
		t.Skipf("only %d pages", len(want))
	}
	r, err := repo.GetPricePlanListPageData(caller, page(1))
	if err != nil {
		t.Fatal(err)
	}
	for i, expected := range want {
		if !slices.Equal(ids(r), expected) {
			t.Fatalf("next page %d: got %v want %v", i+1, ids(r), expected)
		}
		if i+1 < len(want) {
			r, err = repo.GetPricePlanListPageData(caller, cursor(r.GetPagination().GetNextCursor()))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := len(want) - 2; i >= 0; i-- {
		r, err = repo.GetPricePlanListPageData(caller, cursor(r.GetPagination().GetPrevCursor()))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(ids(r), want[i]) {
			t.Fatalf("prev page %d: got %v want %v", i+1, ids(r), want[i])
		}
	}
	foreign, err := repo.GetPricePlanListPageData(caller, cursor("k1:2:next:"+foreignID))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids(foreign), want[1]) {
		t.Fatalf("foreign token escaped offset fallback: got %v want %v", ids(foreign), want[1])
	}
	t.Logf("workspace pages=%d rows=%d, foreign token returned %d caller rows and 0 foreign rows", len(want), r.GetPagination().GetTotalItems(), len(ids(foreign)))
}
