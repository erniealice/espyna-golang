//go:build postgresql

package subscription

import (
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

func TestBuildBalanceListQueriesDefaultsAndTenantScope(t *testing.T) {
	q, err := buildBalanceListQueries(&balancepb.GetBalanceListPageDataRequest{}, "ws-1")
	if err != nil {
		t.Fatalf("buildBalanceListQueries() error = %v", err)
	}
	for _, sql := range []string{q.countSQL, q.dataSQL} {
		for _, fragment := range []string{
			"s.id = b.subscription_id AND s.workspace_id = $1",
			"c.id = b.client_id AND c.workspace_id = $1",
			"WHERE b.active = true",
		} {
			if !strings.Contains(sql, fragment) {
				t.Errorf("SQL missing strict tenant fragment %q:\n%s", fragment, sql)
			}
		}
		if strings.Contains(sql, "$1::text = ''") || strings.Contains(sql, "OR s.workspace_id") {
			t.Errorf("SQL retained empty-workspace bypass:\n%s", sql)
		}
	}
	if !strings.Contains(q.dataSQL, `ORDER BY date_created DESC, "id" ASC LIMIT $2 OFFSET $3`) {
		t.Fatalf("data SQL missing stable bounded page:\n%s", q.dataSQL)
	}
	if q.limit != 50 || q.offset != 0 || q.page != 1 {
		t.Fatalf("bounds = (%d,%d,%d), want (50,0,1)", q.limit, q.offset, q.page)
	}
	assertBalanceArgs(t, q.countArgs, []any{"ws-1"})
	assertBalanceArgs(t, q.dataArgs, []any{"ws-1", int32(50), int32(0)})
}

func TestBuildBalanceListQueriesBoundsAndEscapesSearch(t *testing.T) {
	q, err := buildBalanceListQueries(&balancepb.GetBalanceListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 25,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 2},
			},
		},
		Search: &commonpb.SearchRequest{Query: `50%_off\today`},
	}, "ws-1")
	if err != nil {
		t.Fatalf("buildBalanceListQueries() error = %v", err)
	}
	if q.limit != 25 || q.offset != 25 || q.page != 2 {
		t.Fatalf("bounds = (%d,%d,%d), want (25,25,2)", q.limit, q.offset, q.page)
	}
	if n := strings.Count(q.dataSQL, `ILIKE $`); n != 2 {
		t.Fatalf("search predicate count = %d, want 2:\n%s", n, q.dataSQL)
	}
	if n := strings.Count(q.dataSQL, `ESCAPE '\'`); n != 2 {
		t.Fatalf("explicit LIKE escape count = %d, want 2:\n%s", n, q.dataSQL)
	}
	wantPattern := `%50\%\_off\\today%`
	assertBalanceArgs(t, q.countArgs, []any{"ws-1", wantPattern, wantPattern})
	assertBalanceArgs(t, q.dataArgs, []any{"ws-1", wantPattern, wantPattern, int32(25), int32(25)})
}

func TestBuildBalanceListQueriesRejectsUnboundedOrUnknownInput(t *testing.T) {
	tests := []struct {
		name string
		req  *balancepb.GetBalanceListPageDataRequest
		ws   string
		want string
	}{
		{name: "missing workspace", req: &balancepb.GetBalanceListPageDataRequest{}, want: "workspace is required"},
		{
			name: "oversized page",
			ws:   "ws-1",
			req:  &balancepb.GetBalanceListPageDataRequest{Pagination: &commonpb.PaginationRequest{Limit: 101}},
			want: "pagination limit",
		},
		{
			name: "cursor unsupported",
			ws:   "ws-1",
			req: &balancepb.GetBalanceListPageDataRequest{Pagination: &commonpb.PaginationRequest{
				Limit: 20,
				Method: &commonpb.PaginationRequest_Cursor{
					Cursor: &commonpb.CursorPagination{Token: "offset:20"},
				},
			}},
			want: "cursor pagination is not supported",
		},
		{
			name: "unknown filter",
			ws:   "ws-1",
			req: &balancepb.GetBalanceListPageDataRequest{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
				Field: "password_hash",
				FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
					Operator: commonpb.StringOperator_STRING_EQUALS,
					Value:    "probe",
				}},
			}}}},
			want: "not allowed",
		},
		{
			name: "unknown sort",
			ws:   "ws-1",
			req: &balancepb.GetBalanceListPageDataRequest{Sort: &commonpb.SortRequest{Fields: []*commonpb.SortField{{
				Field: "password_hash",
			}}}},
			want: "unknown sort column",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildBalanceListQueries(tc.req, tc.ws)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestBalanceItemPageDataSQLUsesRelationWorkspaceAnchor(t *testing.T) {
	sql := balanceItemPageDataSQL()
	for _, fragment := range []string{
		"s.id = b.subscription_id AND s.workspace_id = $2",
		"c.id = b.client_id AND c.workspace_id = $2",
		"WHERE b.id = $1 AND b.active = true",
		"LIMIT 1",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("item SQL missing %q:\n%s", fragment, sql)
		}
	}
	if strings.Contains(sql, "$2::text = ''") || strings.Contains(sql, "OR s.workspace_id") {
		t.Fatalf("item SQL retained an empty-workspace bypass:\n%s", sql)
	}
}

func assertBalanceArgs(t *testing.T, got, want []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
