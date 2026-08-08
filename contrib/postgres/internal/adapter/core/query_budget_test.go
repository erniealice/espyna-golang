//go:build postgresql

package core

import (
	"slices"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestValidateListRequestRejectsValidShapedUnknownEntityField(t *testing.T) {
	params := &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "password_hash",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: "probe", Operator: commonpb.StringOperator_STRING_EQUALS,
			}},
		}}},
	}

	err := validateListRequest(params, "client", map[string]struct{}{
		"id": {}, "name": {}, "status": {},
	})
	if err == nil || !strings.Contains(err.Error(), `column "password_hash" is not allowed`) {
		t.Fatalf("validateListRequest() error = %v, want entity-column rejection", err)
	}
}

func TestValidateListRequestRejectsForeignQualifier(t *testing.T) {
	params := &interfaces.ListParams{Sort: &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "other.name"}}}}
	err := validateListRequest(params, "client", map[string]struct{}{"name": {}})
	if err == nil || !strings.Contains(err.Error(), `qualifier "other"`) {
		t.Fatalf("validateListRequest() error = %v, want qualifier rejection", err)
	}
}

func TestValidateListRequestFiniteBudgets(t *testing.T) {
	stringFilter := func(field, value string) *commonpb.TypedFilter {
		return &commonpb.TypedFilter{
			Field: field,
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: value, Operator: commonpb.StringOperator_STRING_CONTAINS,
			}},
		}
	}
	allowed := map[string]struct{}{"name": {}, "status": {}}

	tests := []struct {
		name   string
		params *interfaces.ListParams
		want   string
	}{
		{
			name: "too many filters",
			params: &interfaces.ListParams{Filters: &commonpb.FilterRequest{
				Filters: func() []*commonpb.TypedFilter {
					out := make([]*commonpb.TypedFilter, maxQueryFilters+1)
					for i := range out {
						out[i] = stringFilter("name", "x")
					}
					return out
				}(),
			}},
			want: "too many filters",
		},
		{
			name: "oversized list",
			params: &interfaces.ListParams{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
				Field: "status",
				FilterType: &commonpb.TypedFilter_ListFilter{ListFilter: &commonpb.ListFilter{
					Values: make([]string, maxQueryListValues+1),
				}},
			}}}},
			want: "maximum is 100",
		},
		{
			name: "oversized search",
			params: &interfaces.ListParams{Search: &commonpb.SearchRequest{
				Query:   strings.Repeat("x", maxQuerySearchRunes+1),
				Options: &commonpb.SearchOptions{SearchFields: []string{"name"}},
			}},
			want: "search query exceeds",
		},
		{
			name: "too many search fields",
			params: &interfaces.ListParams{Search: &commonpb.SearchRequest{
				Query:   "x",
				Options: &commonpb.SearchOptions{SearchFields: make([]string, maxQuerySearchFields+1)},
			}},
			want: "too many search fields",
		},
		{
			name: "too many sort fields",
			params: &interfaces.ListParams{Sort: &commonpb.SortRequest{
				Fields: func() []*commonpb.SortField {
					out := make([]*commonpb.SortField, maxQuerySortFields+1)
					for i := range out {
						out[i] = &commonpb.SortField{Field: "name"}
					}
					return out
				}(),
			}},
			want: "too many sort fields",
		},
		{
			name: "oversized page",
			params: &interfaces.ListParams{Pagination: &commonpb.PaginationRequest{
				Limit: maxQueryPageSize + 1,
			}},
			want: "pagination limit",
		},
		{
			name: "oversized offset",
			params: &interfaces.ListParams{Pagination: &commonpb.PaginationRequest{
				Limit: 100,
				Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{
					Page: int32(maxQueryOffset/100) + 2,
				}},
			}},
			want: "pagination offset exceeds",
		},
		{
			name: "malformed cursor",
			params: &interfaces.ListParams{Pagination: &commonpb.PaginationRequest{
				Limit: 20,
				Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{
					Token: "opaque-but-unsupported",
				}},
			}},
			want: "invalid pagination cursor",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateListRequest(tc.params, "client", allowed)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateListRequest() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestBoundedContainsSearchPattern(t *testing.T) {
	tests := []struct {
		name    string
		search  *commonpb.SearchRequest
		want    string
		wantErr string
	}{
		{name: "nil"},
		{name: "empty", search: &commonpb.SearchRequest{}, want: ""},
		{
			name: "unicode rune boundary ignores requested fields",
			search: &commonpb.SearchRequest{
				Query:   strings.Repeat("界", maxQuerySearchRunes),
				Options: &commonpb.SearchOptions{SearchFields: []string{"request_selected_field"}},
			},
			want: "%" + strings.Repeat("界", maxQuerySearchRunes) + "%",
		},
		{
			name:    "too long by rune count",
			search:  &commonpb.SearchRequest{Query: strings.Repeat("界", maxQuerySearchRunes+1)},
			wantErr: "search query exceeds",
		},
		{
			name:   "escapes LIKE wildcards and backslash",
			search: &commonpb.SearchRequest{Query: `a%b_c\d`},
			want:   `%a\%b\_c\\d%`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BoundedContainsSearchPattern(tc.search)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("BoundedContainsSearchPattern() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BoundedContainsSearchPattern() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("BoundedContainsSearchPattern() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoundedLikePatterns(t *testing.T) {
	boundary := strings.Repeat("界", maxQuerySearchRunes)
	tests := []struct {
		name    string
		fn      func(string) (string, error)
		query   string
		want    string
		wantErr string
	}{
		{name: "contains empty", fn: BoundedContainsPattern, want: ""},
		{name: "prefix empty", fn: BoundedPrefixPattern, want: ""},
		{name: "contains unicode boundary", fn: BoundedContainsPattern, query: boundary, want: "%" + boundary + "%"},
		{name: "prefix unicode boundary", fn: BoundedPrefixPattern, query: boundary, want: boundary + "%"},
		{name: "contains escapes wildcards and backslash", fn: BoundedContainsPattern, query: `a%b_c\d`, want: `%a\%b\_c\\d%`},
		{name: "prefix escapes wildcards and backslash", fn: BoundedPrefixPattern, query: `a%b_c\d`, want: `a\%b\_c\\d%`},
		{name: "contains too long", fn: BoundedContainsPattern, query: strings.Repeat("界", maxQuerySearchRunes+1), wantErr: "search query exceeds"},
		{name: "prefix too long", fn: BoundedPrefixPattern, query: strings.Repeat("界", maxQuerySearchRunes+1), wantErr: "search query exceeds"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.fn(tc.query)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("pattern() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("pattern() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("pattern() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoundedQueryLimit(t *testing.T) {
	tests := []struct {
		name                         string
		requested, defaultLimit, max int32
		want                         int32
		wantErr                      string
	}{
		{name: "zero uses default", defaultLimit: 50, max: 100, want: 50},
		{name: "negative uses default", requested: -1, defaultLimit: 50, max: 100, want: 50},
		{name: "within maximum", requested: 75, defaultLimit: 50, max: 100, want: 75},
		{name: "maximum", requested: 100, defaultLimit: 50, max: 100, want: 100},
		{name: "oversize", requested: 101, defaultLimit: 50, max: 100, wantErr: "exceeds maximum"},
		{name: "zero default", defaultLimit: 0, max: 100, wantErr: "invalid query limit configuration"},
		{name: "zero maximum", defaultLimit: 50, max: 0, wantErr: "invalid query limit configuration"},
		{name: "default exceeds maximum", defaultLimit: 101, max: 100, wantErr: "invalid query limit configuration"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BoundedQueryLimit(tc.requested, tc.defaultLimit, tc.max)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("BoundedQueryLimit() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("BoundedQueryLimit() = (%d, %v), want (%d, nil)", got, err, tc.want)
			}
		})
	}
}

func TestBoundedQueryOffset(t *testing.T) {
	tests := []struct {
		name      string
		requested int32
		want      int32
		wantErr   string
	}{
		{name: "negative becomes zero", requested: -1, want: 0},
		{name: "zero", requested: 0, want: 0},
		{name: "maximum", requested: int32(maxQueryOffset), want: int32(maxQueryOffset)},
		{name: "oversize", requested: int32(maxQueryOffset) + 1, wantErr: "exceeds maximum"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BoundedQueryOffset(tc.requested)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("BoundedQueryOffset() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("BoundedQueryOffset() = (%d, %v), want (%d, nil)", got, err, tc.want)
			}
		})
	}
}

func TestListPaginationBoundsCursor(t *testing.T) {
	limit, offset, cursor, err := listPaginationBounds(&commonpb.PaginationRequest{
		Limit: 25,
		Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{
			Token: "offset:75",
		}},
	})
	if err != nil {
		t.Fatalf("listPaginationBounds() unexpected error: %v", err)
	}
	if limit != 25 || offset != 75 || !cursor {
		t.Fatalf("listPaginationBounds() = (%d,%d,%v), want (25,75,true)", limit, offset, cursor)
	}
}

func TestBuildFilterWhereEscapesLiteralWildcards(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "name",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: `50%_off\today`, Operator: commonpb.StringOperator_STRING_CONTAINS,
		}},
	}}}
	search := &commonpb.SearchRequest{Query: `a%b_c\d`}

	clauses, args, next, err := BuildFilterWhere(filters, search, []string{"name"}, 1)
	if err != nil {
		t.Fatalf("BuildFilterWhere() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), `(name ILIKE $1 ESCAPE '\') AND name ILIKE $2 ESCAPE '\'`; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if got, want := args[0], `%a\%b\_c\\d%`; got != want {
		t.Fatalf("search arg = %q, want %q", got, want)
	}
	if got, want := args[1], `%50\%\_off\\today%`; got != want {
		t.Fatalf("filter arg = %q, want %q", got, want)
	}
	if next != 3 {
		t.Fatalf("next = %d, want 3", next)
	}
}

func TestBuildFilterWherePreservesORAndCompoundFilterSemantics(t *testing.T) {
	filters := &commonpb.FilterRequest{
		Logic: commonpb.FilterLogic_OR,
		Filters: []*commonpb.TypedFilter{
			{
				Field: "amount",
				FilterType: &commonpb.TypedFilter_RangeFilter{RangeFilter: &commonpb.RangeFilter{
					Min: 10, Max: 20, IncludeMin: true,
				}},
			},
			{
				Field: "active",
				FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{
					Value: true,
				}},
			},
		},
	}
	search := &commonpb.SearchRequest{Query: "needle"}

	clauses, args, next, err := BuildFilterWhere(filters, search, []string{"name"}, 1)
	if err != nil {
		t.Fatalf("BuildFilterWhere() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), `(name ILIKE $1 ESCAPE '\') AND ((amount >= $2 AND amount < $3) OR active = $4)`; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if len(args) != 4 || args[0] != "%needle%" || args[1] != float64(10) || args[2] != float64(20) || args[3] != true {
		t.Fatalf("args = %#v, want search, range bounds, active", args)
	}
	if next != 5 {
		t.Fatalf("next = %d, want 5", next)
	}
}

func TestGenericFilterBuilderPreservesORAndRangeAsOneArm(t *testing.T) {
	filters := &commonpb.FilterRequest{
		Logic: commonpb.FilterLogic_OR,
		Filters: []*commonpb.TypedFilter{
			{
				Field: "amount",
				FilterType: &commonpb.TypedFilter_RangeFilter{RangeFilter: &commonpb.RangeFilter{
					Min: 1, Max: 5, IncludeMin: true, IncludeMax: true,
				}},
			},
			{
				Field: "active",
				FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{
					Value: true,
				}},
			},
		},
	}

	conditions, args, next, err := (&PostgresOperations{}).buildFilterConditions(filters, 1)
	if err != nil {
		t.Fatalf("buildFilterConditions() unexpected error: %v", err)
	}
	grouped := groupFilterClauses(filters.GetLogic(), conditions)
	if got, want := strings.Join(grouped, " AND "), `((amount >= $1 AND amount <= $2) OR active = $3)`; got != want {
		t.Fatalf("grouped conditions = %q, want %q", got, want)
	}
	if len(args) != 3 || args[0] != float64(1) || args[1] != float64(5) || args[2] != true || next != 4 {
		t.Fatalf("args/next = %#v/%d, want [1 5 true]/4", args, next)
	}
}

func TestBuildFilterWhereImplementsDeclaredStringOperators(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
		{
			Field: "name",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: "EXACT", Operator: commonpb.StringOperator_STRING_NOT_EQUALS,
			}},
		},
		{
			Field: "code",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: "^a.+z$", Operator: commonpb.StringOperator_STRING_REGEX,
			}},
		},
	}}

	clauses, args, next, err := BuildFilterWhere(filters, nil, nil, 3)
	if err != nil {
		t.Fatalf("BuildFilterWhere() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), `LOWER(name) != $3 AND code ~* $4`; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if len(args) != 2 || args[0] != "exact" || args[1] != "^a.+z$" || next != 5 {
		t.Fatalf("BuildFilterWhere() args/next = %#v/%d, want [exact regex]/5", args, next)
	}
}

func TestBuildFilterWhereRejectsFiltersThatWouldOtherwiseDisappear(t *testing.T) {
	tests := []struct {
		name   string
		filter *commonpb.TypedFilter
		want   string
	}{
		{
			name: "empty list",
			filter: &commonpb.TypedFilter{Field: "status", FilterType: &commonpb.TypedFilter_ListFilter{
				ListFilter: &commonpb.ListFilter{},
			}},
			want: "list requires at least one value",
		},
		{
			name: "empty status",
			filter: &commonpb.TypedFilter{Field: "status", FilterType: &commonpb.TypedFilter_StatusFilter{
				StatusFilter: &commonpb.StatusFilter{},
			}},
			want: "status requires at least one value",
		},
		{
			name: "between without end",
			filter: &commonpb.TypedFilter{Field: "date_created", FilterType: &commonpb.TypedFilter_DateFilter{
				DateFilter: &commonpb.DateFilter{Value: "2026-01-01T00:00:00Z", Operator: commonpb.DateOperator_DATE_BETWEEN},
			}},
			want: "DATE_BETWEEN requires range_end",
		},
		{
			name: "unknown string operator",
			filter: &commonpb.TypedFilter{Field: "name", FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{Value: "x", Operator: commonpb.StringOperator(99)},
			}},
			want: "unsupported string operator",
		},
		{
			name: "unknown filter logic",
			filter: &commonpb.TypedFilter{Field: "name", FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{Value: "x", Operator: commonpb.StringOperator_STRING_EQUALS},
			}},
			want: "unsupported filter logic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{tc.filter}}
			if tc.name == "unknown filter logic" {
				request.Logic = commonpb.FilterLogic(99)
			}
			clauses, args, next, err := BuildFilterWhere(request, nil, nil, 7)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("BuildFilterWhere() error = %v, want substring %q", err, tc.want)
			}
			if clauses != nil || args != nil || next != 7 {
				t.Fatalf("BuildFilterWhere() = (%v,%v,%d), want nil,nil,7 on rejection", clauses, args, next)
			}
		})
	}
}

func TestBuildFilterWhereDateBoundaries(t *testing.T) {
	from := "2026-01-01T00:00:00Z"
	to := "2026-02-01T00:00:00Z"
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
		{Field: "date_created", FilterType: &commonpb.TypedFilter_DateFilter{DateFilter: &commonpb.DateFilter{Value: from, Operator: commonpb.DateOperator_DATE_AFTER}}},
		{Field: "date_updated", FilterType: &commonpb.TypedFilter_DateFilter{DateFilter: &commonpb.DateFilter{Value: from, RangeEnd: &to, Operator: commonpb.DateOperator_DATE_BETWEEN}}},
	}}

	clauses, args, next, err := BuildFilterWhere(filters, nil, nil, 3)
	if err != nil {
		t.Fatalf("BuildFilterWhere() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), "date_created >= $3::timestamp AND (date_updated >= $4::timestamp AND date_updated < $5::timestamp)"; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if got, want := args, []any{from, from, to}; !slices.Equal(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
	if next != 6 {
		t.Fatalf("next = %d, want 6", next)
	}
}

func TestBuildFilterWhereAllowedRejectsValidShapedUnknownField(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "password_hash",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "probe", Operator: commonpb.StringOperator_STRING_EQUALS,
		}},
	}}}

	clauses, args, next, err := BuildFilterWhereAllowed(filters, nil, []string{"name", "status"}, nil, 4)
	if err == nil || !strings.Contains(err.Error(), `filter field "password_hash" is not allowed`) {
		t.Fatalf("BuildFilterWhereAllowed() error = %v, want allowlist rejection", err)
	}
	if clauses != nil || args != nil || next != 4 {
		t.Fatalf("BuildFilterWhereAllowed() = (%v,%v,%d), want nil,nil,4 on rejection", clauses, args, next)
	}
}

func TestBuildFilterWhereMappedUsesAdapterOwnedSQLFieldWithoutMutatingRequest(t *testing.T) {
	filter := &commonpb.TypedFilter{
		Field: "status",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "ACTIVE", Operator: commonpb.StringOperator_STRING_EQUALS,
		}},
	}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{filter}}

	clauses, args, next, err := BuildFilterWhereMapped(
		filters,
		nil,
		map[string]string{"status": "rv.status"},
		nil,
		2,
	)
	if err != nil {
		t.Fatalf("BuildFilterWhereMapped() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), "LOWER(rv.status) = $2"; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if len(args) != 1 || args[0] != "active" || next != 3 {
		t.Fatalf("args/next = %#v/%d, want [active]/3", args, next)
	}
	if filter.GetField() != "status" {
		t.Fatalf("request filter mutated to %q", filter.GetField())
	}
}

func TestBuildFilterWhereMappedRejectsUnknownPublicField(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "password_hash",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "probe", Operator: commonpb.StringOperator_STRING_EQUALS,
		}},
	}}}

	clauses, args, next, err := BuildFilterWhereMapped(
		filters, nil, map[string]string{"email_address": "u.email_address"}, nil, 2,
	)
	if err == nil || !strings.Contains(err.Error(), `filter field "password_hash" is not allowed`) {
		t.Fatalf("BuildFilterWhereMapped() error = %v, want allowlist rejection", err)
	}
	if clauses != nil || args != nil || next != 2 {
		t.Fatalf("BuildFilterWhereMapped() = (%v,%v,%d), want nil,nil,2 on rejection", clauses, args, next)
	}
}

func TestBuildFilterWhereMappedISODateTextUsesLexicalDatePredicates(t *testing.T) {
	end := "2026-03-01"
	filter := &commonpb.TypedFilter{
		Field: "payment_date",
		FilterType: &commonpb.TypedFilter_DateFilter{DateFilter: &commonpb.DateFilter{
			Value: "2026-02-01", RangeEnd: &end, Operator: commonpb.DateOperator_DATE_BETWEEN,
		}},
	}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{filter}}

	clauses, args, next, err := BuildFilterWhereMappedISODateText(
		filters,
		nil,
		map[string]string{"payment_date": "tc.payment_date"},
		[]string{"payment_date"},
		nil,
		2,
	)
	if err != nil {
		t.Fatalf("BuildFilterWhereMappedISODateText() unexpected error: %v", err)
	}
	if got, want := strings.Join(clauses, " AND "), "(tc.payment_date >= $2 AND tc.payment_date < $3)"; got != want {
		t.Fatalf("clauses = %q, want %q", got, want)
	}
	if got, want := args, []any{"2026-02-01", "2026-03-01"}; !slices.Equal(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
	if next != 4 {
		t.Fatalf("next = %d, want 4", next)
	}
	if filter.GetField() != "payment_date" {
		t.Fatalf("request filter mutated to %q", filter.GetField())
	}
}

func TestBuildFilterWhereMappedISODateTextRejectsMalformedDate(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "payment_date",
		FilterType: &commonpb.TypedFilter_DateFilter{DateFilter: &commonpb.DateFilter{
			Value: "2026-99-01", Operator: commonpb.DateOperator_DATE_BEFORE,
		}},
	}}}

	clauses, args, next, err := BuildFilterWhereMappedISODateText(
		filters,
		nil,
		map[string]string{"payment_date": "d.payment_date"},
		[]string{"payment_date"},
		nil,
		7,
	)
	if err == nil || !strings.Contains(err.Error(), "valid YYYY-MM-DD date") {
		t.Fatalf("error = %v, want strict ISO-date rejection", err)
	}
	if clauses != nil || args != nil || next != 7 {
		t.Fatalf("result = (%v,%v,%d), want nil,nil,7 on rejection", clauses, args, next)
	}
}

func TestBoundedOffsetPaginationRejectsUnboundedInput(t *testing.T) {
	tests := []struct {
		name string
		p    *commonpb.PaginationRequest
	}{
		{name: "too large", p: &commonpb.PaginationRequest{Limit: 101}},
		{name: "negative", p: &commonpb.PaginationRequest{Limit: -1}},
		{name: "cursor", p: &commonpb.PaginationRequest{
			Limit:  20,
			Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: "offset:0"}},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := BoundedOffsetPagination(tc.p, 50); err == nil {
				t.Fatal("BoundedOffsetPagination() error = nil, want rejection")
			}
		})
	}
}

func TestBoundedOffsetPaginationUsesDefaultForProtoZeroLimit(t *testing.T) {
	limit, offset, page, err := BoundedOffsetPagination(&commonpb.PaginationRequest{}, 50)
	if err != nil {
		t.Fatalf("BoundedOffsetPagination() unexpected error: %v", err)
	}
	if limit != 50 || offset != 0 || page != 1 {
		t.Fatalf("BoundedOffsetPagination() = (%d,%d,%d), want (50,0,1)", limit, offset, page)
	}
}

func TestBoundedQueryIDsDeduplicatesWithinBudget(t *testing.T) {
	got, err := BoundedQueryIDs([]string{"client-2", "client-1", "client-2"})
	if err != nil {
		t.Fatalf("BoundedQueryIDs() unexpected error: %v", err)
	}
	if want := []string{"client-2", "client-1"}; !slices.Equal(got, want) {
		t.Fatalf("BoundedQueryIDs() = %v, want %v", got, want)
	}
}

func TestBoundedQueryIDsRejectsAbusiveInput(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{name: "too many", values: make([]string, maxQueryListValues+1)},
		{name: "empty", values: []string{"client-1", ""}},
		{name: "oversized", values: []string{strings.Repeat("x", maxQueryFilterValueRunes+1)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BoundedQueryIDs(tc.values); err == nil {
				t.Fatal("BoundedQueryIDs() error = nil, want rejection")
			}
		})
	}
}
