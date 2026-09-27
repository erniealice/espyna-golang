//go:build postgresql

package core

import (
	"fmt"
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestBoundedPageRequest_ModesAndFallback(t *testing.T) {
	const id = "12345678-1234-1234-1234-123456789abc"
	request := func(token string) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 7, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}
	}
	for _, tc := range []struct {
		token        string
		mode         PageMode
		page, offset int32
	}{
		{"k1:3:next:" + id, PageModeKeyset, 3, 14},
		{"k1:2:prev:" + id, PageModeKeyset, 2, 7},
		{"offset:14", PageModeOffset, 3, 14},
		{"garbled", PageModeOffset, 1, 0},
		{"k1:0:next:" + id, PageModeOffset, 1, 0},
		{strings.Repeat("x", 65), PageModeOffset, 1, 0},
	} {
		got, err := BoundedPageRequest(request(tc.token), 50)
		if err != nil || got.Mode != tc.mode || got.Number != tc.page || got.Offset != tc.offset || got.Limit != 7 {
			t.Errorf("token %q: %+v, %v", tc.token, got, err)
		}
	}
	got, err := BoundedPageRequest(&commonpb.PaginationRequest{Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 2}}}, 50)
	if err != nil || got.Limit != 50 || got.Offset != 50 || got.Number != 2 {
		t.Fatalf("default limit: %+v, %v", got, err)
	}
	if _, _, _, err := BoundedOffsetPagination(request("k1:2:next:"+id), 50); err == nil {
		t.Fatal("offset-only API accepted a cursor")
	}
}

func TestScopedPageQueries_ScopeAndPlaceholders(t *testing.T) {
	const id = "12345678-1234-1234-1234-123456789abc"
	for _, argCount := range []int{0, 3, 6} {
		t.Run(fmt.Sprint(argCount), func(t *testing.T) {
			predicates := []string{"active = true"}
			args := make([]any, argCount)
			for i := range args {
				predicates = append(predicates, fmt.Sprintf("scope_%d = $%d", i+1, i+1))
				args[i] = i
			}
			set := ScopedPageSet{
				SQL:  "SELECT id, score FROM demo WHERE " + strings.Join(predicates, " AND "),
				Args: args, Sort: []AdapterSortKey{{Column: "score", Desc: true, NullsFirst: false}},
			}
			keys, err := validateScopedPage(set, Page{Mode: PageModeKeyset, Limit: 7, Number: 2, Offset: 7, BoundaryID: id, Direction: "next"})
			if err != nil {
				t.Fatal(err)
			}
			lookup, lookupArgs := scopedPageBoundarySQL(set, keys, id)
			if !strings.Contains(lookup, "FROM ("+set.SQL+") b WHERE b.id = $"+fmt.Sprint(argCount+1)) || len(lookupArgs) != argCount+1 || lookupArgs[argCount] != id {
				t.Fatalf("boundary: %s / %v", lookup, lookupArgs)
			}
			page := buildScopedPageQueries(set, keys, Page{Mode: PageModeKeyset, Limit: 7, Number: 2, Offset: 7, BoundaryID: id, Direction: "next"}, []any{int64(5), id})
			for _, want := range []string{set.SQL, `"score" < $` + fmt.Sprint(argCount+1), `"id" > $` + fmt.Sprint(argCount+2), "LIMIT $" + fmt.Sprint(argCount+3), "FROM scoped b", "JOIN page_keys k"} {
				if !strings.Contains(page.PageSQL, want) {
					t.Errorf("page SQL lacks %q: %s", want, page.PageSQL)
				}
			}
			if len(page.PageArgs) != argCount+3 || len(page.CountArgs) != argCount || page.CountSQL != "SELECT COUNT(*) FROM ("+set.SQL+") b" {
				t.Fatalf("args/count: %+v", page)
			}
			offset := buildScopedPageQueries(set, keys, Page{Mode: PageModeOffset, Limit: 7, Number: 2, Offset: 7}, nil)
			if !strings.Contains(offset.PageSQL, fmt.Sprintf("LIMIT $%d OFFSET $%d", argCount+1, argCount+2)) || len(offset.PageArgs) != argCount+2 {
				t.Fatalf("offset SQL/args: %+v", offset)
			}
		})
	}
}

func TestScopedPageQueries_NullsReverseTiesAndIdentifierGuard(t *testing.T) {
	const id = "12345678-1234-1234-1234-123456789abc"
	for _, nullsFirst := range []bool{true, false} {
		for _, direction := range []string{"next", "prev"} {
			set := ScopedPageSet{SQL: "SELECT id, grade FROM demo", Sort: []AdapterSortKey{{Column: "grade", NullsFirst: nullsFirst}}}
			page := Page{Mode: PageModeKeyset, Limit: 7, Number: 2, Offset: 7, BoundaryID: id, Direction: direction}
			keys, err := validateScopedPage(set, page)
			if err != nil {
				t.Fatal(err)
			}
			built := buildScopedPageQueries(set, keys, page, []any{nil, id})
			if !strings.Contains(built.PageSQL, `"id"`) || !strings.Contains(built.PageSQL, "IS NOT DISTINCT FROM $1") {
				t.Fatalf("missing tie/null equality: %s", built.PageSQL)
			}
			if direction == "next" && nullsFirst && !strings.Contains(built.PageSQL, `"grade" IS NOT NULL`) {
				t.Fatalf("nulls-first next lacks non-null arm: %s", built.PageSQL)
			}
			if direction == "prev" && nullsFirst && strings.Contains(built.PageSQL, `"grade" IS NOT NULL`) {
				t.Fatalf("nulls-first reverse has wrong non-null arm: %s", built.PageSQL)
			}
		}
	}
	for _, column := range []string{"x; DROP TABLE demo", "other.grade"} {
		if _, err := adapterSortKeys([]AdapterSortKey{{Column: column}}); err == nil {
			t.Fatalf("accepted unsafe sort column %q", column)
		}
	}
}
