//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"github.com/lib/pq"
)

func keysetParityIDs(t *testing.T, db *sql.DB, table, workspace string, nullable bool) []string {
	t.Helper()
	var query string
	if nullable {
		query = `SELECT id FROM (
            (SELECT id FROM job WHERE active AND workspace_id = $1 AND parent_job_id IS NULL ORDER BY id LIMIT 20)
            UNION ALL
            (SELECT id FROM job WHERE active AND workspace_id = $1 AND parent_job_id IS NOT NULL ORDER BY id LIMIT 20)
        ) sampled ORDER BY id`
	} else {
		query = fmt.Sprintf(`SELECT id FROM "%s" WHERE active AND workspace_id = $1
            AND date_created = (SELECT date_created FROM "%s" WHERE active AND workspace_id = $1
                GROUP BY date_created ORDER BY count(*) DESC LIMIT 1)
            ORDER BY id LIMIT 40`, table, table)
	}
	rows, err := db.QueryContext(context.Background(), query, workspace)
	if err != nil {
		t.Fatalf("sample %s: %v", table, err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) < 20 {
		t.Skipf("%s has only %d sampled rows", table, len(ids))
	}
	return ids
}

func keysetParityParams(ids []string, sort *commonpb.SortRequest, page int32, token string) *interfaces.ListParams {
	p := &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "id", FilterType: &commonpb.TypedFilter_ListFilter{ListFilter: &commonpb.ListFilter{
				Values: ids, Operator: commonpb.ListOperator_LIST_IN,
			}},
		}}},
		Sort:       sort,
		Pagination: &commonpb.PaginationRequest{Limit: 7},
	}
	if token != "" {
		p.Pagination.Method = &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}
	} else {
		p.Pagination.Method = &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}
	}
	return p
}

func listIDs(result *interfaces.ListResult) []string {
	ids := make([]string, 0, len(result.Data))
	for _, row := range result.Data {
		ids = append(ids, fmt.Sprint(row["id"]))
	}
	return ids
}

// TestListParity_KeysetWalk proves Next and Prev against the offset oracle on
// three live tables. The job sample crosses NULL and non-NULL parent_job_id;
// phase/task samples have 40 identical date_created values each.
func TestListParity_KeysetWalk(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()
	p := &PostgresOperations{db: db}
	cases := []struct {
		name     string
		table    string
		nullable bool
		sort     *commonpb.SortRequest
	}{
		{"job_nulls_first", "job", true, &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "parent_job_id", Direction: commonpb.SortDirection_ASC, NullOrder: commonpb.NullOrder_NULLS_FIRST}}}},
		{"job_nulls_last", "job", true, &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "parent_job_id", Direction: commonpb.SortDirection_ASC, NullOrder: commonpb.NullOrder_NULLS_LAST}}}},
		{"job_phase", "job_phase", false, &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "date_created", Direction: commonpb.SortDirection_DESC, NullOrder: commonpb.NullOrder_NULLS_LAST}}}},
		{"job_task", "job_task", false, &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "date_created", Direction: commonpb.SortDirection_ASC, NullOrder: commonpb.NullOrder_NULLS_FIRST}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var workspace string
			if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT workspace_id FROM "%s" WHERE active LIMIT 1`, tc.table)).Scan(&workspace); err != nil {
				t.Skipf("no workspace sample: %v", err)
			}
			ids := keysetParityIDs(t, db, tc.table, workspace, tc.nullable)
			scope := []*commonpb.TypedFilter{{Field: "workspace_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: workspace, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}}}
			var pages [][]string
			var responses []*interfaces.ListResult
			for page := int32(1); ; page++ {
				result, err := p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, page, ""))
				if err != nil {
					t.Fatalf("offset page %d: %v", page, err)
				}
				pages = append(pages, listIDs(result))
				responses = append(responses, result)
				if !result.Pagination.GetHasNext() {
					break
				}
			}
			if len(pages) < 3 {
				t.Fatalf("want at least 3 pages, got %d", len(pages))
			}
			t.Logf("sampled_ids=%d offset_pages=%d page_size=7", len(ids), len(pages))
			for i, result := range responses {
				if result.Pagination.GetCurrentPage() != int32(i+1) || result.Pagination.GetTotalPages() != int32(len(pages)) {
					t.Fatalf("offset metadata on page %d: %+v", i+1, result.Pagination)
				}
			}
			// Start through the cursor path, then walk Next to the end.
			result, err := p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 1, "offset:0"))
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < len(pages); i++ {
				if !slices.Equal(listIDs(result), pages[i]) || result.Pagination.GetCurrentPage() != int32(i+1) || result.Pagination.GetTotalPages() != int32(len(pages)) {
					t.Fatalf("next page %d: got %v, want %v", i+1, listIDs(result), pages[i])
				}
				if i+1 == len(pages) {
					break
				}
				result, err = p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 0, result.Pagination.GetNextCursor()))
				if err != nil {
					t.Fatalf("next page %d: %v", i+2, err)
				}
			}
			for i := len(pages) - 1; i >= 0; i-- {
				if !slices.Equal(listIDs(result), pages[i]) {
					t.Fatalf("prev page %d: got %v, want %v", i+1, listIDs(result), pages[i])
				}
				if i == 0 {
					break
				}
				result, err = p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 0, result.Pagination.GetPrevCursor()))
				if err != nil {
					t.Fatalf("prev page %d: %v", i, err)
				}
			}
			// Page-number navigation remains the bounded offset route. Its
			// response must seed a keyset Next cursor for the following click.
			jump := responses[2]
			if !slices.Equal(listIDs(jump), pages[2]) || jump.Pagination.GetNextCursor() == "" {
				t.Fatalf("offset jump page 3 has no usable next cursor")
			}
			if len(pages) > 3 {
				afterJump, err := p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 0, jump.Pagination.GetNextCursor()))
				if err != nil || !slices.Equal(listIDs(afterJump), pages[3]) {
					t.Fatalf("next after offset jump: ids=%v err=%v", listIDs(afterJump), err)
				}
			}
			// A boundary outside the caller filter must use the requested page's
			// offset path. This is the same lookup used for workspace scope.
			var outOfFilter string
			// The token decoder requires a UUID-shaped boundary. Some local
			// workspaces also contain legacy human-readable ids (for example
			// job-001), which would make this a malformed-token test instead.
			if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT id FROM "%s" WHERE active AND workspace_id = $1
				AND NOT (id = ANY($2))
				AND id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
				ORDER BY id LIMIT 1`, tc.table), workspace, pq.Array(ids)).Scan(&outOfFilter); err != nil {
				t.Fatal(err)
			}
			if !uuidShaped(outOfFilter) {
				t.Fatalf("out-of-filter boundary %q is not UUID-shaped", outOfFilter)
			}
			forged := encodeKeysetToken(2, "next", outOfFilter)
			result, err = p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 0, forged))
			if err != nil {
				t.Fatalf("out-of-filter page-two fallback: %v", err)
			}
			if !slices.Equal(listIDs(result), pages[1]) || result.Pagination.GetCurrentPage() != 2 {
				t.Fatalf("out-of-filter page-two fallback: got=%v want=%v page=%d", listIDs(result), pages[1], result.Pagination.GetCurrentPage())
			}
			malformed, err := p.ListWithScope(ctx, tc.table, scope, keysetParityParams(ids, tc.sort, 0, "garbled"))
			if err != nil || !slices.Equal(listIDs(malformed), pages[0]) {
				t.Fatalf("malformed page-one fallback: ids=%v err=%v", listIDs(malformed), err)
			}
			// The clone has one workspace. Simulate a foreign workspace scope
			// against a real row id and prove the scoped PK lookup misses it.
			foreignScope := []*commonpb.TypedFilter{{Field: "workspace_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: "00000000-0000-0000-0000-000000000001", Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}}}
			foreignWhere := "active = true AND workspace_id = $1"
			boundary, found, err := p.scopedBoundary(ctx, tc.table, foreignWhere, []any{"00000000-0000-0000-0000-000000000001"}, listSortKeys(keysetParityParams(ids, tc.sort, 1, "")), ids[0])
			if err != nil || found || boundary != nil {
				t.Fatalf("foreign-workspace boundary = (%v,%t,%v), want absent", boundary, found, err)
			}
			foreign, err := p.ListWithScope(ctx, tc.table, foreignScope, keysetParityParams(ids, tc.sort, 0, encodeKeysetToken(1, "next", ids[0])))
			if err != nil || len(foreign.Data) != 0 || foreign.Pagination.GetCurrentPage() != 1 {
				t.Fatalf("foreign scope result = (%v,%v)", foreign, err)
			}
		})
	}
}

// TestListParity_KeysetForeignWorkspace uses two populated workspaces when the
// local clone provides them. A valid token naming a real foreign row must not
// reposition the caller; it returns the caller's first offset page instead.
func TestListParity_KeysetForeignWorkspace(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, `SELECT workspace_id FROM product WHERE active GROUP BY workspace_id ORDER BY count(*) DESC, workspace_id LIMIT 2`)
	if err != nil {
		t.Fatalf("workspace sample: %v", err)
	}
	var workspaces []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		workspaces = append(workspaces, id)
	}
	rows.Close()
	if len(workspaces) < 2 {
		t.Skip("clone has fewer than two populated product workspaces")
	}
	var foreignID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM product WHERE active AND workspace_id = $1 ORDER BY id LIMIT 1`, workspaces[1]).Scan(&foreignID); err != nil {
		t.Fatal(err)
	}
	if !uuidShaped(foreignID) {
		t.Fatalf("foreign product id is not UUID-shaped")
	}
	scope := []*commonpb.TypedFilter{{Field: "workspace_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: workspaces[0], Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}}}
	sort := &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id", Direction: commonpb.SortDirection_ASC, NullOrder: commonpb.NullOrder_NULLS_LAST}}}
	params := func(token string) *interfaces.ListParams {
		p := &interfaces.ListParams{Sort: sort, Pagination: &commonpb.PaginationRequest{Limit: 7}}
		if token == "" {
			p.Pagination.Method = &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 1}}
		} else {
			p.Pagination.Method = &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}
		}
		return p
	}
	op := &PostgresOperations{db: db}
	first, err := op.ListWithScope(ctx, "product", scope, params(""))
	if err != nil {
		t.Fatal(err)
	}
	forged, err := op.ListWithScope(ctx, "product", scope, params(encodeKeysetToken(1, "next", foreignID)))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(listIDs(forged), listIDs(first)) || forged.Pagination.GetCurrentPage() != 1 {
		t.Fatalf("foreign row changed page: first=%v forged=%v page=%d", listIDs(first), listIDs(forged), forged.Pagination.GetCurrentPage())
	}
	for _, id := range listIDs(forged) {
		if id == foreignID {
			t.Fatal("foreign product row escaped scope")
		}
	}
	var foreignRows int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM product WHERE id = ANY($1) AND workspace_id <> $2`, pq.Array(listIDs(forged)), workspaces[0]).Scan(&foreignRows); err != nil {
		t.Fatal(err)
	}
	if foreignRows != 0 {
		t.Fatalf("forged page contains %d foreign workspace rows", foreignRows)
	}
	t.Logf("foreign-workspace token fell back to page one; rows=%d", len(forged.Data))
}
