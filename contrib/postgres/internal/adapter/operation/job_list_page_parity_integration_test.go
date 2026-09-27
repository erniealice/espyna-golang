//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

// Plan 20260927-db-query-performance AC-09 (audit DB-07), pilot on the job list.
// GetJobListPageData moved from COUNT(*) OVER () + full-width sort to an
// ids-first page with a scalar count. This compares the adapter's ids, order,
// and total with the frozen legacy statement. SELECT-only; skips without
// TEST_DATABASE_URL.

// legacyJobListPageSQL is the pre-change statement shape (ids + total only; the
// projection does not affect which rows or in what order).
func legacyJobListPageSQL(jobScope, orderByClause string) string {
	return fmt.Sprintf(`
		WITH enriched AS (
			SELECT j.id, j.date_created, j.date_modified, j.name, j.status
			FROM `+entityid.Job+` j
			WHERE j.active = true
			  AND ($1 = '' OR j.workspace_id = $1)
			  AND ($2::text IS NULL OR $2::text = '' OR
			       j.name ILIKE $2)%s
		)
		SELECT e.id, COUNT(*) OVER () AS total
		FROM enriched e
		%s
		LIMIT $3 OFFSET $4;
	`, jobScope, orderByClause)
}

func TestJobListPageDB_Parity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()

	var ws string
	if err := db.QueryRowContext(ctx,
		`SELECT workspace_id FROM `+entityid.Job+` GROUP BY workspace_id ORDER BY COUNT(*) DESC, workspace_id LIMIT 1`,
	).Scan(&ws); err != nil {
		t.Skipf("no jobs: %v", err)
	}
	var staffID string
	_ = db.QueryRowContext(ctx,
		`SELECT staff_id FROM `+entityid.SubscriptionSeat+`
		 WHERE status = 'active' AND active AND workspace_id = $1
		 GROUP BY staff_id ORDER BY COUNT(*) DESC, staff_id LIMIT 1`, ws).Scan(&staffID)

	repo := NewPostgresJobRepository(postgresCore.NewWorkspaceAwareOperations(db), "job").(*PostgresJobRepository)
	page := func(n int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 50, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}
	}
	byName := &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "e.name", Direction: commonpb.SortDirection_ASC}}}

	cases := []struct {
		name  string
		req   *pb.GetJobListPageDataRequest
		staff bool
	}{
		{"first_page", &pb.GetJobListPageDataRequest{Pagination: page(1)}, false},
		{"deep_page", &pb.GetJobListPageDataRequest{Pagination: page(201)}, false},
		{"past_the_end", &pb.GetJobListPageDataRequest{Pagination: page(5000)}, false},
		{"search", &pb.GetJobListPageDataRequest{Pagination: page(1), Search: &commonpb.SearchRequest{Query: "a"}}, false},
		{"sort_by_name_page_3", &pb.GetJobListPageDataRequest{Pagination: page(3), Sort: byName}, false},
		{"staff_first_page", &pb.GetJobListPageDataRequest{Pagination: page(1)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rid := &identity.RequestIdentity{WorkspaceID: ws}
			if tc.staff {
				if staffID == "" {
					t.Skip("no seat staff sample")
				}
				rid.PrincipalType, rid.PrincipalID = 7, staffID
			}
			reqCtx := identity.WithRequestIdentity(ctx, rid)

			searchPattern, err := postgresCore.BoundedContainsSearchPattern(tc.req.GetSearch())
			if err != nil {
				t.Fatal(err)
			}
			limit, offset, _, err := postgresCore.BoundedOffsetPagination(tc.req.GetPagination(), 50)
			if err != nil {
				t.Fatal(err)
			}
			orderBy, err := postgresCore.BuildOrderBy(jobSortableSQLCols, tc.req.GetSort(), "e.date_created DESC")
			if err != nil {
				t.Fatal(err)
			}
			scope, scopeArgs := principalscope.StaffReachableJobClause(reqCtx, "j", 5)
			args := append([]any{ws, searchPattern, limit, offset}, scopeArgs...)

			rows, err := db.QueryContext(ctx, legacyJobListPageSQL(scope, orderBy), args...)
			if err != nil {
				t.Fatalf("legacy query: %v", err)
			}
			var wantIDs []string
			var wantTotal int64
			for rows.Next() {
				var id string
				if err := rows.Scan(&id, &wantTotal); err != nil {
					t.Fatal(err)
				}
				wantIDs = append(wantIDs, id)
			}
			rows.Close()

			resp, err := repo.GetJobListPageData(reqCtx, tc.req)
			if err != nil {
				t.Fatalf("GetJobListPageData: %v", err)
			}
			var gotIDs []string
			for _, j := range resp.GetJobList() {
				gotIDs = append(gotIDs, j.GetId())
			}
			if !slices.Equal(gotIDs, wantIDs) {
				t.Fatalf("ids/order differ: new=%d rows, legacy=%d rows", len(gotIDs), len(wantIDs))
			}
			// The legacy window cannot report a total for an empty page; only
			// compare totals when the legacy page has rows.
			if len(wantIDs) > 0 && int64(resp.GetPagination().GetTotalItems()) != wantTotal {
				t.Fatalf("total: new=%d legacy=%d", resp.GetPagination().GetTotalItems(), wantTotal)
			}
			t.Logf("rows=%d total=%d", len(gotIDs), resp.GetPagination().GetTotalItems())
		})
	}
}
