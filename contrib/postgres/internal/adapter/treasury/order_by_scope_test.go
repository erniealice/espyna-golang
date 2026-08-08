//go:build postgresql

package treasury

// Finding-3 close-out shape test: the disbursement list adapter orders the OUTER
// `SELECT e.* FROM enriched e` subquery, so its sort whitelist + map + fallback must
// reference the outer alias `e`, never the inner `d`. Locks the emitted ORDER BY as
// outer-scoped, per-component quoted, terminating in the id tiebreaker.

import (
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func TestTreasuryCashEventListOrderByOuterScope(t *testing.T) {
	sortReq := func(col string, dir commonpb.SortDirection) *commonpb.SortRequest {
		return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: col, Direction: dir}}}
	}
	targets := []struct {
		name     string
		allowed  []string
		fieldMap map[string]string
	}{
		{"collection", collectionSortableSQLCols, collectionViewToSQLColMap},
		{"disbursement", disbursementSortableSQLCols, disbursementViewToSQLColMap},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			cases := []struct {
				name string
				sort *commonpb.SortRequest
				want string
			}{
				{"fallback", nil, `ORDER BY e.date_created DESC, "id" ASC`},
				{"public sort", sortReq("amount", commonpb.SortDirection_DESC), `ORDER BY "e"."amount" DESC, "id" ASC`},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					got, err := postgresCore.BuildOrderBy(
						target.allowed,
						mapTreasurySortRequest(tc.sort, target.fieldMap),
						"e.date_created DESC",
					)
					if err != nil {
						t.Fatalf("BuildOrderBy: %v", err)
					}
					if got != tc.want {
						t.Errorf("got %q, want %q", got, tc.want)
					}
					if strings.Contains(got, `"d".`) || strings.Contains(got, `"tc".`) {
						t.Errorf("emitted clause leaks an inner alias: %q", got)
					}
				})
			}
		})
	}
}

func TestMapTreasurySortRequestCopiesAndRejectsUnknownFields(t *testing.T) {
	original := &commonpb.SortRequest{Fields: []*commonpb.SortField{
		{Field: "amount", Direction: commonpb.SortDirection_DESC},
	}}

	mapped := mapTreasurySortRequest(original, collectionViewToSQLColMap)
	if got := mapped.GetFields()[0].GetField(); got != "e.amount" {
		t.Fatalf("mapped field = %q, want e.amount", got)
	}
	if got := original.GetFields()[0].GetField(); got != "amount" {
		t.Fatalf("mapping mutated caller request: got %q", got)
	}

	unknown := mapTreasurySortRequest(sortRequest("amount; DROP TABLE treasury_collection", commonpb.SortDirection_ASC), collectionViewToSQLColMap)
	if _, err := postgresCore.BuildOrderBy(collectionSortableSQLCols, unknown, "e.date_created DESC"); err == nil {
		t.Fatal("BuildOrderBy accepted an unknown/injection-shaped sort field")
	}
}

func sortRequest(field string, direction commonpb.SortDirection) *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: field, Direction: direction}}}
}

func TestTreasuryCashEventFiltersKeepPlaceholdersBoundAndTextDatesComparable(t *testing.T) {
	end := "2026-09-01"
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "payment_date",
		FilterType: &commonpb.TypedFilter_DateFilter{DateFilter: &commonpb.DateFilter{
			Value: "2026-08-01", RangeEnd: &end, Operator: commonpb.DateOperator_DATE_BETWEEN,
		}},
	}}}
	search := &commonpb.SearchRequest{Query: "tuition%_probe"}

	targets := []struct {
		name         string
		fieldMap     map[string]string
		searchFields []string
		wantDate     string
	}{
		{"collection", collectionFilterFieldMap, []string{"tc.name", "tc.reference_number", "tc.status", "tc.collection_type"}, "tc.payment_date"},
		{"disbursement", disbursementFilterFieldMap, []string{"d.name", "d.reference_number", "d.status", "d.disbursement_type"}, "d.payment_date"},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			clauses, args, next, err := postgresCore.BuildFilterWhereMappedISODateText(
				filters, search, target.fieldMap, []string{"payment_date"}, target.searchFields, 2,
			)
			if err != nil {
				t.Fatalf("BuildFilterWhereMappedISODateText: %v", err)
			}
			joined := strings.Join(clauses, " AND ")
			wantDate := "(" + target.wantDate + " >= $6 AND " + target.wantDate + " < $7)"
			if !strings.Contains(joined, wantDate) {
				t.Fatalf("clauses = %q, want date clause %q", joined, wantDate)
			}
			if strings.Contains(joined, "::timestamp") {
				t.Fatalf("TEXT payment_date compared as timestamp: %q", joined)
			}
			if len(args) != 6 || next != 8 {
				t.Fatalf("args/next = %d/%d, want 6/8 (LIMIT $8 OFFSET $9)", len(args), next)
			}
			if got, ok := args[0].(string); !ok || got != `%tuition\%\_probe%` {
				t.Fatalf("search argument = %#v, want escaped wildcard literal", args[0])
			}
		})
	}
}

func TestTreasuryCashEventFiltersRejectUnknownFieldsBeforeQuery(t *testing.T) {
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "workspace_id; DROP TABLE workspace",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "probe", Operator: commonpb.StringOperator_STRING_EQUALS,
		}},
	}}}
	if _, _, _, err := postgresCore.BuildFilterWhereMappedISODateText(
		filters, nil, collectionFilterFieldMap, []string{"payment_date"}, nil, 2,
	); err == nil {
		t.Fatal("unknown/injection-shaped filter field was accepted")
	}
}

func TestRequireTreasuryRawDBRejectsMissingConnection(t *testing.T) {
	if err := requireTreasuryRawDB(nil); err == nil {
		t.Fatal("nil raw PostgreSQL connection was accepted")
	}
}
